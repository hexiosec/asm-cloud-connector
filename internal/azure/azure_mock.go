package azure

import (
	"context"
	"testing"

	cloud_provider_t "github.com/hexiosec/asm-cloud-connector/internal/cloud_provider/types"
	"github.com/stretchr/testify/mock"
)

type MockWrapper struct {
	mock.Mock
}

func NewMockWrapper(t *testing.T) IAzureWrapper {
	t.Helper()
	m := &MockWrapper{}
	m.Mock.Test(t)
	t.Cleanup(func() {
		t.Helper()
		m.AssertExpectations(t)
	})
	return m
}

func (m *MockWrapper) CheckConnection(_ context.Context) error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWrapper) InitResourceGraph(_ context.Context) error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWrapper) GetPublicIPs(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetPublicIPDNSNames(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetApplicationGatewayHostnames(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetApplicationGatewayCertificateDomains(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetFrontDoorClassicHostnames(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetFrontDoorAfdHostnames(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetTrafficManagerFQDNs(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetDNSZones(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetDNSRecordFQDNs(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetStorageWebEndpoints(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetCDNEndpointHostnames(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetAppServiceHostnames(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetSQLServerFQDNs(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetCosmosDocumentEndpoints(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetRedisHostnames(_ context.Context) ([]*cloud_provider_t.Asset, error) {
	args := m.Called()
	return getAssetSlice(args.Get(0)), args.Error(1)
}

func getAssetSlice(value interface{}) []*cloud_provider_t.Asset {
	if value == nil {
		return nil
	}
	return value.([]*cloud_provider_t.Asset)
}
