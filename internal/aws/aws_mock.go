package aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

type MockWrapper struct {
	mock.Mock
}

func NewMockWrapper(t *testing.T) IAWSWrapper {
	t.Helper()
	m := &MockWrapper{}
	m.Mock.Test(t)
	t.Cleanup(func() {
		t.Helper()
		m.AssertExpectations(t)
	})
	return m
}

func (m *MockWrapper) AssumeRole(_ context.Context, role string) (IAWSWrapper, error) {
	args := m.Called(role)
	if wrapper := args.Get(0); wrapper != nil {
		return wrapper.(IAWSWrapper), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWrapper) ChangeRegion(region string) {
	m.Called(region)
}

func (m *MockWrapper) ResetRegion() {
	m.Called()
}

func (m *MockWrapper) CheckConnection(_ context.Context) error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWrapper) GetSecretString(_ context.Context, secret string) (string, error) {
	args := m.Called(secret)
	if value := args.Get(0); value != nil {
		return value.(string), args.Error(1)
	}
	return "", args.Error(1)
}

func (m *MockWrapper) ListAllAccounts(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetRegions(_ context.Context) ([]string, error) {
	args := m.Called()
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetEC2Resources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetEIPResources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetELBResources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetS3Resources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetACMResources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetRoute53Resources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetCloudFrontResources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetAPIGatewayResources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetAPIGatewayV2Resources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetEKSResources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetRDSResources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetOpenSearchResources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func (m *MockWrapper) GetLambdaResources(_ context.Context, resources []string) ([]string, error) {
	args := m.Called(resources)
	return getStringSlice(args.Get(0)), args.Error(1)
}

func getStringSlice(value interface{}) []string {
	if value == nil {
		return nil
	}
	return value.([]string)
}
