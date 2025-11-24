package cloud_provider_t

import (
	"context"
	"fmt"
)

var ErrNoAPIKey = fmt.Errorf("no API key")

type CloudProvider interface {
	Authenticate(ctx context.Context) error
	GetResources(ctx context.Context) ([]string, error)
	GetAPIKey(ctx context.Context) (string, error)
}
