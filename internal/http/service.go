package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
)

// Implementation of resty's Logger interface mapped to our logger
type requestLogger struct{}

func (*requestLogger) Errorf(format string, v ...interface{}) {
	logger.GetGlobalLogger().Error().Bool("is_resty_err", true).Msgf(format, v...)
}
func (*requestLogger) Warnf(format string, v ...interface{}) {
	logger.GetGlobalLogger().Warn().Bool("is_resty_err", true).Msgf(format, v...)
}
func (*requestLogger) Debugf(format string, v ...interface{}) {
	logger.GetGlobalLogger().Debug().Bool("is_resty_err", true).Msgf(format, v...)
}

type IHttpService interface {
	Get(ctx context.Context, url string, options HttpOptions) (IHttpResponse, error)
}

type HttpService struct {
	client    *resty.Client
	userAgent string
}

func NewHttpService(config *config.Config, userAgent string) IHttpService {
	client := resty.New()

	// Configure automatic retry
	client.
		SetLogger(&requestLogger{}).
		SetRetryCount(config.Http.RetryCount).          // Maximum retries
		SetRetryWaitTime(config.Http.RetryBaseDelay).   // Initial backoff time
		SetRetryMaxWaitTime(config.Http.RetryMaxDelay). // Maximum backoff time
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				// Request failed with no response, likely recoverable (i.e. network error)
				if err != nil {
					return true
				}

				// 429 Too Many Requests should be recoverable with the retry and backoff
				if r.StatusCode() == http.StatusTooManyRequests {
					return true
				}

				if r.StatusCode() >= 500 && r.StatusCode() != http.StatusNotImplemented {
					logger.GetLogger(r.Request.Context()).Debug().Int("status_code", r.StatusCode()).Msg("Unexpected HTTP status, retrying")
					return true
				}

				return false
			},
		)

	return &HttpService{
		client:    client,
		userAgent: userAgent,
	}
}

// Get performs a GET request to the given URL.
func (s *HttpService) Get(ctx context.Context, url string, options HttpOptions) (IHttpResponse, error) {
	req := s.client.R()

	req.SetHeaders(options.Headers)
	req.SetHeader("User-Agent", s.userAgent)
	req.SetQueryParams(options.QueryParams)
	req.SetContext(ctx)

	httpRes, err := req.Get(url)
	if err != nil {
		return nil, err
	}

	contentType := strings.ToLower(httpRes.Header().Get("Content-Type"))

	var body interface{}
	if strings.HasPrefix(contentType, "application/json") {
		if err := json.Unmarshal(httpRes.Body(), &body); err != nil {
			return nil, err
		}
	} else if strings.HasPrefix(contentType, "text/plain") {
		// Look for header prefix to make sure 'text/plain; charset=utf-8' is captured
		body = string(httpRes.Body())
	}

	return &HttpResponse{
		StatusCode: httpRes.StatusCode(),
		RawBody:    httpRes.Body(),
		Body:       body,
		Header:     httpRes.Header(),
	}, nil
}
