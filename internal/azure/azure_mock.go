package azure

import (
	"context"
	"testing"

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

func (m *MockWrapper) GetPublicIPs(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetPublicIPDNSNames(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetApplicationGatewayHostnames(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetApplicationGatewayCertificateDomains(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetFrontDoorClassicHostnames(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetFrontDoorAfdHostnames(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetTrafficManagerFQDNs(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetDNSZones(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetDNSRecordFQDNs(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetStorageWebEndpoints(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetCDNEndpointHostnames(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetAppServiceHostnames(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetSQLServerFQDNs(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetCosmosDocumentEndpoints(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetRedisHostnames(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func getStringSlice(value interface{}) []string {
	if value == nil {
		return nil
	}
	return value.([]string)
}
