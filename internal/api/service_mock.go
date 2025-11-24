package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/hexiosec/asm-sdk-go"
	"github.com/stretchr/testify/mock"
)

type MockAPI struct {
	mock.Mock
}

func NewMockAPI(t *testing.T) API {
	t.Helper()
	m := &MockAPI{}
	m.Mock.Test(t)
	t.Cleanup(func() {
		t.Helper()
		m.AssertExpectations(t)
	})
	return m
}

func (m *MockAPI) GetState(ctx context.Context) (*asm.AuthResponse, *http.Response, error) {
	args := m.Called()

	var auth *asm.AuthResponse
	if v := args.Get(0); v != nil {
		auth = v.(*asm.AuthResponse)
	}

	var resp *http.Response
	if v := args.Get(1); v != nil {
		resp = v.(*http.Response)
	}

	return auth, resp, args.Error(2)
}

func (m *MockAPI) GetScanByID(ctx context.Context, scanID string) (*asm.ScanResponse, *http.Response, error) {
	args := m.Called(scanID)

	var scan *asm.ScanResponse
	if v := args.Get(0); v != nil {
		scan = v.(*asm.ScanResponse)
	}

	var resp *http.Response
	if v := args.Get(1); v != nil {
		resp = v.(*http.Response)
	}

	return scan, resp, args.Error(2)
}

func (m *MockAPI) GetScanSeedsById(ctx context.Context, scanID string) ([]asm.SeedsResponseInner, *http.Response, error) {
	args := m.Called(scanID)

	var seeds []asm.SeedsResponseInner
	if v := args.Get(0); v != nil {
		seeds = v.([]asm.SeedsResponseInner)
	}

	var resp *http.Response
	if v := args.Get(1); v != nil {
		resp = v.(*http.Response)
	}

	return seeds, resp, args.Error(2)
}

func (m *MockAPI) AddScanSeedById(ctx context.Context, scanID string, request asm.CreateScanSeedRequest) (*asm.NodeResponse, *http.Response, error) {
	args := m.Called(scanID, request)

	var node *asm.NodeResponse
	if v := args.Get(0); v != nil {
		node = v.(*asm.NodeResponse)
	}

	var resp *http.Response
	if v := args.Get(1); v != nil {
		resp = v.(*http.Response)
	}

	return node, resp, args.Error(2)
}

func (m *MockAPI) RemoveScanSeedById(ctx context.Context, scanID string, seedID string) (*http.Response, error) {
	args := m.Called(scanID, seedID)

	var resp *http.Response
	if v := args.Get(0); v != nil {
		resp = v.(*http.Response)
	}

	return resp, args.Error(1)
}
