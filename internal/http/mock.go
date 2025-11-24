package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

type MockHttpService struct {
	mock.Mock
}

func NewMockHttpService(t *testing.T) IHttpService {
	t.Helper()
	s := MockHttpService{}
	s.Mock.Test(t)
	t.Cleanup(func() {
		t.Helper()
		s.AssertExpectations(t)
	})
	return &s
}

func (m *MockHttpService) Get(ctx context.Context, url string, options HttpOptions) (IHttpResponse, error) {
	args := m.Called(url, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MockHttpResponse), args.Error(1)
}
