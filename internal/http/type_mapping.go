package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
)

type HttpOptions struct {
	Headers     map[string]string
	QueryParams map[string]string
}

type IHttpResponse interface {
	GetStatusCode() int
	HasBody() bool
	GetBody() interface{}
	GetRawBody() []byte
	GetHeader() http.Header
}

type HttpResponse struct {
	StatusCode int
	Body       interface{}
	RawBody    []byte
	Header     http.Header
}

func (r *HttpResponse) GetStatusCode() int {
	return r.StatusCode
}

func (r *HttpResponse) HasBody() bool {
	return len(r.RawBody) > 0
}

func (r *HttpResponse) GetBody() interface{} {
	return r.Body
}

func (r *HttpResponse) GetRawBody() []byte {
	return r.RawBody
}

func (r *HttpResponse) GetHeader() http.Header {
	return r.Header
}

type MockHttpResponse struct {
	mock.Mock
}

func NewMockHttpResponse(t *testing.T) *MockHttpResponse {
	t.Helper()
	m := &MockHttpResponse{}
	m.Mock.Test(t)
	t.Cleanup(func() {
		t.Helper()
		m.AssertExpectations(t)
	})
	return m
}

func (m *MockHttpResponse) GetStatusCode() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockHttpResponse) HasBody() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockHttpResponse) GetBody() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockHttpResponse) GetRawBody() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
}

func (m *MockHttpResponse) GetHeader() http.Header {
	args := m.Called()
	return args.Get(0).(http.Header)
}
