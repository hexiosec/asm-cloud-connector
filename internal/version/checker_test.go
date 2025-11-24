package version

import (
	"bytes"
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hexiosec/asm-cloud-connector/internal/http"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
)

type dependencies struct {
	ctx       context.Context
	http      *http.MockHttpService
	logBuffer *bytes.Buffer
}

func newTestChecker(t *testing.T) (*checker, *dependencies) {
	t.Helper()
	buf := &bytes.Buffer{}
	log := zerolog.New(buf)
	ctx := logger.WithLogger(context.Background(), log)

	deps := dependencies{
		ctx:       ctx,
		http:      http.NewMockHttpService(t).(*http.MockHttpService),
		logBuffer: buf,
	}
	return &checker{http: deps.http}, &deps
}

func TestLogVersion_GetLatestVersionErr_Warns(t *testing.T) {
	checker, deps := newTestChecker(t)

	deps.http.On(
		"Get",
		home,
		mock.Anything,
	).Return(nil, assert.AnError)

	checker.LogVersion(deps.ctx)

	assert.Contains(t, deps.logBuffer.String(), "Failed to get latest version")
}

func TestLogVersion_NoRemoteVersions_AssumesLatest(t *testing.T) {
	checker, deps := newTestChecker(t)

	resp := http.NewMockHttpResponse(t)
	resp.On("GetStatusCode").Return(404)
	deps.http.On(
		"Get",
		home,
		mock.Anything,
	).Return(resp, nil)

	checker.LogVersion(deps.ctx)

	assert.Contains(t, deps.logBuffer.String(), "assuming latest version")
}

func TestLogVersion_RemoteGreater_Warns(t *testing.T) {
	checker, deps := newTestChecker(t)

	resp := http.NewMockHttpResponse(t)
	resp.On("GetStatusCode").Return(200)
	resp.On("HasBody").Return(true)
	resp.On("GetBody").Return(map[string]any{
		"tag_name": "v1.2.3",
	})
	deps.http.On(
		"Get",
		home,
		mock.Anything,
	).Return(resp, nil)

	checker.LogVersion(deps.ctx)

	assert.Contains(t, deps.logBuffer.String(), "New version available")
}

func TestLogVersion_RemoteNotGreater_ReportsLatest(t *testing.T) {
	checker, deps := newTestChecker(t)

	resp := http.NewMockHttpResponse(t)
	resp.On("GetStatusCode").Return(200)
	resp.On("HasBody").Return(true)
	resp.On("GetBody").Return(map[string]any{
		"tag_name": "v0.0.0",
	})
	deps.http.On(
		"Get",
		home,
		mock.Anything,
	).Return(resp, nil)

	checker.LogVersion(deps.ctx)

	assert.Contains(t, deps.logBuffer.String(), "Running latest version")
}

func TestLogVersion_InvalidRemoteVersion_WarnsCompare(t *testing.T) {
	checker, deps := newTestChecker(t)

	resp := http.NewMockHttpResponse(t)
	resp.On("GetStatusCode").Return(200)
	resp.On("HasBody").Return(true)
	resp.On("GetBody").Return(map[string]any{
		"tag_name": "invalid",
	})
	deps.http.On(
		"Get",
		home,
		mock.Anything,
	).Return(resp, nil)

	checker.LogVersion(deps.ctx)

	assert.Contains(t, deps.logBuffer.String(), "Failed to compare")
}

func TestGetLatestVersion_HTTPError_Err(t *testing.T) {
	checker, deps := newTestChecker(t)

	deps.http.On(
		"Get",
		home,
		mock.Anything,
	).Return(nil, assert.AnError)

	ok, version, err := checker.getLatestVersion(deps.ctx)

	assert.False(t, ok)
	assert.Empty(t, version)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestGetLatestVersion_Non200_Err(t *testing.T) {
	checker, deps := newTestChecker(t)

	resp := http.NewMockHttpResponse(t)
	resp.On("GetStatusCode").Return(400)
	deps.http.On(
		"Get",
		home,
		mock.Anything,
	).Return(resp, nil)

	ok, version, err := checker.getLatestVersion(deps.ctx)

	assert.False(t, ok)
	assert.Empty(t, version)
	assert.Contains(t, err.Error(), "400")
}

func TestGetLatestVersion_NoBody_Err(t *testing.T) {
	checker, deps := newTestChecker(t)

	resp := http.NewMockHttpResponse(t)
	resp.On("GetStatusCode").Return(200)
	resp.On("HasBody").Return(false)
	deps.http.On(
		"Get",
		home,
		mock.Anything,
	).Return(resp, nil)

	ok, version, err := checker.getLatestVersion(deps.ctx)

	assert.False(t, ok)
	assert.Empty(t, version)
	assert.Contains(t, err.Error(), "no body")
}

func TestGetLatestVersion_DecodeError_Err(t *testing.T) {
	checker, deps := newTestChecker(t)

	resp := http.NewMockHttpResponse(t)
	resp.On("GetStatusCode").Return(200)
	resp.On("HasBody").Return(true)
	resp.On("GetBody").Return(map[string]any{
		"NotName": "v1.2.3",
	})
	deps.http.On(
		"Get",
		home,
		mock.Anything,
	).Return(resp, nil)

	ok, version, err := checker.getLatestVersion(deps.ctx)

	assert.False(t, ok)
	assert.Empty(t, version)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to destruct and validate response")
}

func TestGetLatestVersion_ReleaseNotFound_False(t *testing.T) {
	checker, deps := newTestChecker(t)

	resp := http.NewMockHttpResponse(t)
	resp.On("GetStatusCode").Return(404)
	deps.http.On(
		"Get",
		home,
		mock.Anything,
	).Return(resp, nil)

	ok, version, err := checker.getLatestVersion(deps.ctx)

	assert.False(t, ok)
	assert.Empty(t, version)
	assert.NoError(t, err)
}

func TestGetLatestVersion_Release_True(t *testing.T) {
	checker, deps := newTestChecker(t)

	resp := http.NewMockHttpResponse(t)
	resp.On("GetStatusCode").Return(200)
	resp.On("HasBody").Return(true)
	resp.On("GetBody").Return(map[string]any{
		"tag_name": "v1.2.3",
	})
	deps.http.On(
		"Get",
		home,
		mock.Anything,
	).Return(resp, nil)

	ok, version, err := checker.getLatestVersion(deps.ctx)

	assert.True(t, ok)
	assert.Equal(t, "v1.2.3", version)
	assert.NoError(t, err)
}

func TestIsGreaterThan_RemoteGreater_True(t *testing.T) {
	ok, err := isGreaterThan("v1.0.1", "v1.0.0")

	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestIsGreaterThan_RemoteGreater_NoV_True(t *testing.T) {
	ok, err := isGreaterThan("1.0.1", "1.0.0")

	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestIsGreaterThan_InvalidRemote_Error(t *testing.T) {
	ok, err := isGreaterThan("invalid", "v1.0.0")

	assert.Error(t, err)
	assert.False(t, ok)
}

func TestIsGreaterThan_InvalidCurrent_Error(t *testing.T) {
	ok, err := isGreaterThan("v1.0.0", "invalid")

	assert.Error(t, err)
	assert.False(t, ok)
}

func TestIsGreaterThan_RemoteNotGreater_False(t *testing.T) {
	ok, err := isGreaterThan("v1.0.0", "v1.0.1")

	assert.NoError(t, err)
	assert.False(t, ok)
}
