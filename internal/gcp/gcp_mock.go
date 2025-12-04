package gcp

import (
	"context"
	"testing"

	"cloud.google.com/go/asset/apiv1/assetpb"
	certificatemanagerpb "cloud.google.com/go/certificatemanager/apiv1/certificatemanagerpb"
	"github.com/stretchr/testify/mock"
)

type MockWrapper struct {
	mock.Mock
}

func NewMockWrapper(t *testing.T) IGCPWrapper {
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

func (m *MockWrapper) GetAssets(_ context.Context, project string, assetTypes []string) ([]*assetpb.Asset, error) {
	args := m.Called(project, assetTypes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*assetpb.Asset), args.Error(1)
}

func (m *MockWrapper) GetCertificates(_ context.Context, project string) ([]*certificatemanagerpb.Certificate, error) {
	args := m.Called(project)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*certificatemanagerpb.Certificate), args.Error(1)
}

func (m *MockWrapper) IsBucketPublic(_ context.Context, bucketName string) bool {
	args := m.Called(bucketName)
	if val := args.Get(0); val != nil {
		return val.(bool)
	}
	return false
}
