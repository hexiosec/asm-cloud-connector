package connector

import (
	"context"
	"testing"

	"github.com/hexiosec/asm-cloud-connector/internal/config"
	asm "github.com/hexiosec/asm-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnector_InvalidIgnoreList(t *testing.T) {
	for _, entry := range []string{"", "   ", "not a domain", "://", "https://", "*.example.com", "example.*", "203.0.113.0/24", "2001:db8::/32"} {
		t.Run(entry, func(t *testing.T) {
			conn, err := NewConnector(&config.Config{IgnoreList: []string{entry}}, nil)
			require.ErrorContains(t, err, "ignore_list[0]")
			assert.Nil(t, conn)
		})
	}
}

func TestSyncResources_IgnoreListMatchesNormalisedSeeds(t *testing.T) {
	cfg := &config.Config{
		ScanID:  "scan-123",
		SeedTag: "seed-tag",
		IgnoreList: []string{
			" https://THIRD-PARTY.Example.COM.:443/path ",
			"https://203.0.113.10:8443/path",
			"third-party.example.com", // Duplicate entries are harmless.
		},
	}
	conn, mockAPI := newTestConnector(t, cfg)
	mockAPI.On("GetScanSeedsById", cfg.ScanID).
		Return([]asm.SeedsResponseInner{}, nil, nil).Once()

	// Matching is exact: subdomains and similar names remain eligible.
	for _, name := range []string{"sub.third-party.example.com", "other-third-party.example.com", "203.0.113.11"} {
		mockAPI.On("AddScanSeedById", cfg.ScanID, asm.CreateScanSeedRequest{
			Name: name,
			Type: getResourceType(name),
			Tags: []string{cfg.SeedTag},
		}).Return(&asm.NodeResponse{}, nil, nil).Once()
	}

	err := conn.SyncResources(context.Background(), assetSlice(
		assetWithTags("third-party.example.com", "provider-tag"),
		asset("https://Third-Party.Example.COM/path"),
		asset("*.third-party.example.com."),
		asset("203.0.113.10"),
		asset("sub.third-party.example.com"),
		asset("other-third-party.example.com"),
		asset("203.0.113.11"),
	))
	require.NoError(t, err)
}

func TestSyncResources_IgnoredExistingSeeds(t *testing.T) {
	for _, deleteStale := range []bool{false, true} {
		name := "keep existing seeds"
		if deleteStale {
			name = "remove only connector seeds"
		}
		t.Run(name, func(t *testing.T) {
			cfg := &config.Config{
				ScanID:           "scan-123",
				SeedTag:          "seed-tag",
				DeleteStaleSeeds: deleteStale,
				IgnoreList:       []string{"ignored.example.com", "manual.example.com"},
			}
			conn, mockAPI := newTestConnector(t, cfg)
			mockAPI.On("GetScanSeedsById", cfg.ScanID).
				Return([]asm.SeedsResponseInner{
					{Name: "ignored.example.com", Id: "connector-id", Tags: []string{cfg.SeedTag}},
					{Name: "manual.example.com", Id: "manual-id", Tags: []string{"other-tag"}},
				}, nil, nil).Once()
			if deleteStale {
				mockAPI.On("RemoveScanSeedById", cfg.ScanID, "connector-id").Return(nil, nil).Once()
			}

			err := conn.SyncResources(context.Background(), assetSlice(
				asset("https://ignored.example.com/path"),
				asset("manual.example.com"),
			))
			require.NoError(t, err)
		})
	}
}
