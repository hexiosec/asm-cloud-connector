package connector

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hexiosec/asm-cloud-connector/internal/api"
	cloud_provider_t "github.com/hexiosec/asm-cloud-connector/internal/cloud_provider/types"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	asm "github.com/hexiosec/asm-sdk-go"
)

func newTestConnector(t *testing.T, cfg *config.Config) (*Connector, *api.MockAPI) {
	t.Helper()
	mockAPI := api.NewMockAPI(t).(*api.MockAPI)
	conn, err := NewConnector(cfg, mockAPI)
	assert.NoError(t, err)
	return conn, mockAPI
}

func TestConnector_Authenticate_Success(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "tag",
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetState").
		Return(&asm.AuthResponse{Authenticated: true}, nil, nil)
	mockAPI.On("GetScanByID", cfg.ScanID).
		Return(&asm.ScanResponse{}, nil, nil)

	err := conn.Authenticate(context.Background())
	assert.NoError(t, err)
}

func TestConnector_Authenticate_NotAuthenticated_Err(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "tag",
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetState").
		Return(&asm.AuthResponse{Authenticated: false}, nil, nil)

	err := conn.Authenticate(context.Background())
	assert.ErrorContains(t, err, "credentials not valid")
}

func TestConnector_Authenticate_ScanErr_Err(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "tag",
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetState").
		Return(&asm.AuthResponse{Authenticated: true}, nil, nil)
	mockAPI.On("GetScanByID", cfg.ScanID).
		Return(nil, nil, assert.AnError)

	err := conn.Authenticate(context.Background())
	assert.Error(t, err)
	assert.ErrorAs(t, err, &assert.AnError)
}

func TestSyncResources_Normalise_Success(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "seed-tag",
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetScanSeedsById", cfg.ScanID).
		Return([]asm.SeedsResponseInner{}, nil, nil)

	mockAPI.On("AddScanSeedById", cfg.ScanID, mock.MatchedBy(func(req asm.CreateScanSeedRequest) bool {
		expected := asm.CreateScanSeedRequest{
			Name: "example.com",
			Type: resourceDomain,
			Tags: []string{cfg.SeedTag},
		}
		return reflect.DeepEqual(req, expected)
	})).
		Return(&asm.NodeResponse{}, nil, nil).
		Once()

	mockAPI.On("AddScanSeedById", cfg.ScanID, mock.MatchedBy(func(req asm.CreateScanSeedRequest) bool {
		expected := asm.CreateScanSeedRequest{
			Name: "example2.com",
			Type: resourceDomain,
			Tags: []string{cfg.SeedTag},
		}
		return reflect.DeepEqual(req, expected)
	})).
		Return(&asm.NodeResponse{}, nil, nil).
		Once()

	err := conn.SyncResources(context.Background(), assetSlice(
		asset("example.com"),
		asset("https://Example.COM "),
		asset("example2.com"),
		asset("example.com"),
	))
	assert.NoError(t, err)
}

func TestSyncResources_ExistingSeed_Skipped(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "seed-tag",
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetScanSeedsById", cfg.ScanID).
		Return([]asm.SeedsResponseInner{
			{Name: "existing.com"},
		}, nil, nil)

	mockAPI.On("AddScanSeedById", cfg.ScanID, mock.MatchedBy(func(req asm.CreateScanSeedRequest) bool {
		expected := asm.CreateScanSeedRequest{
			Name: "example.com",
			Type: resourceDomain,
			Tags: []string{cfg.SeedTag},
		}
		return reflect.DeepEqual(req, expected)
	})).
		Return(&asm.NodeResponse{}, nil, nil).
		Once()

	err := conn.SyncResources(context.Background(), assetSlice(
		asset("example.com"),
		asset("existing.com"),
	))
	assert.NoError(t, err)
}

func TestSyncResources_GetSeedsErr_Err(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "seed-tag",
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetScanSeedsById", cfg.ScanID).Return(nil, nil, assert.AnError)

	err := conn.SyncResources(context.Background(), assetSlice(
		asset("example.com"),
		asset("existing.com"),
	))
	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestSyncResources_IPv6Resource_Skipped(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "seed-tag",
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetScanSeedsById", cfg.ScanID).
		Return([]asm.SeedsResponseInner{}, nil, nil)

	err := conn.SyncResources(context.Background(), assetSlice(
		asset("2001:0db8:85a3:0000:0000:8a2e:0370:7334"),
		asset("2345:0425:2CA1::0567:5673:23b5"),
	))
	assert.NoError(t, err)
}

func TestSyncResources_AddSeed_500_Err(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "seed-tag",
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetScanSeedsById", cfg.ScanID).
		Return([]asm.SeedsResponseInner{}, nil, nil)

	mockAPI.On("AddScanSeedById", cfg.ScanID, mock.Anything).Return(nil, &http.Response{StatusCode: 500}, assert.AnError)

	err := conn.SyncResources(context.Background(), assetSlice(asset("example.com")))
	assert.Error(t, err)
	assert.ErrorAs(t, err, &assert.AnError)
}

func TestSyncResources_AddSeed_400WithValidBody_NonFatalCase_Skipped(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "seed-tag",
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetScanSeedsById", cfg.ScanID).
		Return([]asm.SeedsResponseInner{}, nil, nil)

	body := `{"code":"ERR123"}`
	mockAPI.On("AddScanSeedById", cfg.ScanID, mock.Anything).Return(
		nil,
		&http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(body))},
		assert.AnError,
	)

	err := conn.SyncResources(context.Background(), assetSlice(asset("example.com")))
	assert.NoError(t, err)
}

func TestSyncResources_AddSeed_400WithInValidBody_FatalCase_Err(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "seed-tag",
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetScanSeedsById", cfg.ScanID).
		Return([]asm.SeedsResponseInner{}, nil, nil)

	body := `{}`
	mockAPI.On("AddScanSeedById", cfg.ScanID, mock.Anything).Return(
		nil,
		&http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(body))},
		assert.AnError,
	)

	err := conn.SyncResources(context.Background(), assetSlice(asset("example.com")))
	assert.Error(t, err)
	assert.ErrorAs(t, err, &assert.AnError)
}

func TestSyncResources_RemovesStaleSeedsWithTag_Success(t *testing.T) {
	cfg := &config.Config{
		ScanID:           "scan-123",
		SeedTag:          "seed-tag",
		DeleteStaleSeeds: true,
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetScanSeedsById", cfg.ScanID).
		Return([]asm.SeedsResponseInner{
			{Name: "keep.com", Tags: []string{cfg.SeedTag}, Id: "keep-id"},
			{Name: "stale.com", Tags: []string{cfg.SeedTag}, Id: "stale-id"},
			{Name: "skip.com", Tags: []string{"other-tag"}, Id: "skip-id"},
		}, nil, nil)

	mockAPI.On("RemoveScanSeedById", cfg.ScanID, "stale-id").
		Return(&http.Response{}, nil).
		Once()

	err := conn.SyncResources(context.Background(), assetSlice(asset("keep.com")))
	assert.NoError(t, err)
}

func TestSyncResources_DeleteSeedFails_Continue(t *testing.T) {
	cfg := &config.Config{
		ScanID:           "scan-123",
		SeedTag:          "seed-tag",
		DeleteStaleSeeds: true,
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetScanSeedsById", cfg.ScanID).
		Return([]asm.SeedsResponseInner{
			{Name: "keep.com", Tags: []string{cfg.SeedTag}, Id: "keep-id"},
			{Name: "stale.com", Tags: []string{cfg.SeedTag}, Id: "stale-id"},
			{Name: "stale2.com", Tags: []string{cfg.SeedTag}, Id: "stale2-id"},
			{Name: "skip.com", Tags: []string{"other-tag"}, Id: "skip-id"},
		}, nil, nil)

	mockAPI.On("RemoveScanSeedById", cfg.ScanID, "stale-id").
		Return(nil, assert.AnError).
		Once()

	mockAPI.On("RemoveScanSeedById", cfg.ScanID, "stale2-id").
		Return(nil, assert.AnError).
		Once()

	err := conn.SyncResources(context.Background(), assetSlice(asset("keep.com")))
	assert.NoError(t, err)
}

func TestDedup(t *testing.T) {
	tests := []struct {
		name     string
		input    []*cloud_provider_t.Asset
		expected []*cloud_provider_t.Asset
	}{
		{
			name:     "NilSlice_ReturnsNil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "SingleItem_Unchanged",
			input:    assetSlice(asset("a")),
			expected: assetSlice(asset("a")),
		},
		{
			name:     "NoDuplicates_Unchanged",
			input:    assetSlice(asset("a"), asset("b"), asset("c")),
			expected: assetSlice(asset("a"), asset("b"), asset("c")),
		},
		{
			name:  "Duplicates_MergeTags",
			input: assetSlice(assetWithTags("a", "one"), assetWithTags("a", "two"), asset("b"), assetWithTags("b", "three")),
			expected: assetSlice(
				assetWithTags("a", "one", "two"),
				assetWithTags("b", "three"),
			),
		},
		{
			name:     "NilAndEmptyNames_Skipped",
			input:    assetSlice(nil, asset(""), asset("x")),
			expected: assetSlice(asset("x")),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var inputCopy []*cloud_provider_t.Asset
			if tc.input != nil {
				inputCopy = make([]*cloud_provider_t.Asset, len(tc.input))
				copy(inputCopy, tc.input)
			}

			got := dedup(context.Background(), inputCopy)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestNormalise(t *testing.T) {
	input := assetSlice(
		assetWithTags("http://Example.com/path", "one"),
		asset("https://user:pass@Sub.Domain.COM:8443/anything?query=1"),
		asset(" 192.168.0.1 "),
		asset("2001:db8::1"),
		asset("[2001:db8::1]"),
		asset("not a domain"),
		asset("https://[2001:db8::2]:443/foo"),
		asset("ftp://user@host.example.org"),
		asset("*.cloudrun.regr.creepycrawly.io."),
	)

	got := normalise(context.Background(), input)

	expected := assetSlice(
		assetWithTags("example.com", "one"),
		asset("sub.domain.com"),
		asset("192.168.0.1"),
		asset("2001:db8::1"),
		asset("2001:db8::1"),
		asset("2001:db8::2"),
		asset("host.example.org"),
		asset("cloudrun.regr.creepycrawly.io"),
	)

	assert.Equal(t, expected, got)
}

func TestNormalise_Invalid_LogsWarn(t *testing.T) {
	var buf bytes.Buffer

	prevLogger := log.Logger
	log.Logger = zerolog.New(&buf)
	t.Cleanup(func() { log.Logger = prevLogger })

	got := normalise(context.Background(), assetSlice(
		nil,
		asset(""),
		asset("://"),
		asset("???"),
		asset("ftp://"),
		asset("."),
		asset("-example.com"),
	))

	assert.Empty(t, got)
	assert.Contains(t, buf.String(), "Unable to normalise resource")
}

func TestSyncResources_MergesAssetTagsBeforeCreate(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "seed-tag",
	}
	conn, mockAPI := newTestConnector(t, cfg)

	mockAPI.On("GetScanSeedsById", cfg.ScanID).
		Return([]asm.SeedsResponseInner{}, nil, nil)

	mockAPI.On("AddScanSeedById", cfg.ScanID, mock.MatchedBy(func(req asm.CreateScanSeedRequest) bool {
		expected := asm.CreateScanSeedRequest{
			Name: "example.com",
			Type: resourceDomain,
			Tags: []string{"alpha", "beta", cfg.SeedTag},
		}
		return reflect.DeepEqual(req, expected)
	})).
		Return(&asm.NodeResponse{}, nil, nil).
		Once()

	err := conn.SyncResources(context.Background(), assetSlice(
		assetWithTags("Example.com", "beta"),
		assetWithTags("https://example.com", "alpha", "beta"),
	))
	assert.NoError(t, err)
}

func asset(name string) *cloud_provider_t.Asset {
	return &cloud_provider_t.Asset{Name: name}
}

func assetWithTags(name string, tags ...string) *cloud_provider_t.Asset {
	return &cloud_provider_t.Asset{Name: name, Tags: tags}
}

func assetSlice(assets ...*cloud_provider_t.Asset) []*cloud_provider_t.Asset {
	return assets
}

func TestGetErrorCode_Success(t *testing.T) {
	input := `{"code":"ERR123"}`
	body := io.NopCloser(strings.NewReader(input))

	code, err := getErrorCode(body)
	assert.NoError(t, err)
	assert.Equal(t, "ERR123", code)
}

func TestGetErrorCode_Invalid_Err(t *testing.T) {
	input := `{"not_a_code":"ERR123"}`
	body := io.NopCloser(strings.NewReader(input))

	code, err := getErrorCode(body)
	assert.Error(t, err)
	assert.Empty(t, code)
}

func TestGetErrorCode_NotJSON_Err(t *testing.T) {
	input := ``
	body := io.NopCloser(strings.NewReader(input))

	code, err := getErrorCode(body)
	assert.Error(t, err)
	assert.Empty(t, code)
}

func TestGetErrorCode_OtherFields_Success(t *testing.T) {
	input := `{"code":"ERR123","other":"exists"}`
	body := io.NopCloser(strings.NewReader(input))

	code, err := getErrorCode(body)
	assert.NoError(t, err)
	assert.Equal(t, "ERR123", code)
}
