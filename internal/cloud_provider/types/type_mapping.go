package cloud_provider_t

import (
	"context"
	"fmt"
)

var ErrNoAPIKey = fmt.Errorf("no API key")

type Asset struct {
	Name string
	Tags []string
}

type CloudProvider interface {
	Authenticate(ctx context.Context) error
	GetResources(ctx context.Context) ([]*Asset, error)
	GetAPIKey(ctx context.Context) (string, error)
	GetName() string
}
