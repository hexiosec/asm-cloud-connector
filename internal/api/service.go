package api

import (
	"context"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
	"github.com/hexiosec/asm-sdk-go"
)

type API interface {
	GetState(ctx context.Context) (*asm.AuthResponse, *http.Response, error)
	GetScanByID(ctx context.Context, scanID string) (*asm.ScanResponse, *http.Response, error)
	GetScanSeedsById(ctx context.Context, scanID string) ([]asm.SeedsResponseInner, *http.Response, error)
	AddScanSeedById(ctx context.Context, scanID string, request asm.CreateScanSeedRequest) (*asm.NodeResponse, *http.Response, error)
	RemoveScanSeedById(ctx context.Context, scanID string, seedID string) (*http.Response, error)
}

type sdk struct {
	client *asm.APIClient
}

func NewAPI(cfg *config.Config, userAgent string, apiKey string) (API, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = cfg.Http.RetryCount
	retryClient.RetryWaitMax = cfg.Http.RetryMaxDelay
	retryClient.RetryWaitMin = cfg.Http.RetryBaseDelay
	retryClient.Logger = &logger.RetryableLogger{}

	sdkCfg := asm.NewConfiguration()
	sdkCfg.HTTPClient = retryClient.StandardClient()
	sdkCfg.UserAgent = userAgent
	sdkCfg.APIKey = apiKey

	return &sdk{client: asm.NewAPIClient(sdkCfg)}, nil
}

func (s *sdk) GetState(ctx context.Context) (*asm.AuthResponse, *http.Response, error) {
	return s.client.AuthAPI.GetState(ctx).Execute()
}

func (s *sdk) GetScanByID(ctx context.Context, scanID string) (*asm.ScanResponse, *http.Response, error) {
	return s.client.ScansAPI.GetScanByID(ctx, scanID).Execute()
}

func (s *sdk) GetScanSeedsById(ctx context.Context, scanID string) ([]asm.SeedsResponseInner, *http.Response, error) {
	return s.client.ScansAPI.GetScanSeedsById(ctx, scanID).Expand([]string{"tags"}).Execute()
}

func (s *sdk) AddScanSeedById(ctx context.Context, scanID string, request asm.CreateScanSeedRequest) (*asm.NodeResponse, *http.Response, error) {
	return s.client.ScansAPI.AddScanSeedById(ctx, scanID).CreateScanSeedRequest(request).Execute()
}

func (s *sdk) RemoveScanSeedById(ctx context.Context, scanID string, seedID string) (*http.Response, error) {
	return s.client.ScansAPI.RemoveScanSeedById(ctx, scanID, seedID).Execute()
}
