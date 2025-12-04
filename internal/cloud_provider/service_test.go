package cloud_provider

import (
	"testing"

	"github.com/hexiosec/asm-cloud-connector/internal/aws"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
	"github.com/hexiosec/asm-cloud-connector/internal/gcp"
	"github.com/stretchr/testify/assert"
)

func TestNewCloudProvider_AWSEnabled_Success(t *testing.T) {
	cfg := &config.Config{
		AWS: &config.AWSCloudProvider{
			CloudProvider: config.CloudProvider{Enabled: true},
		},
	}

	provider, err := NewCloudProvider(cfg)

	assert.NoError(t, err)
	assert.IsType(t, &aws.AWSProvider{}, provider)
}

func TestNewCloudProvider_AzureEnabled_ErrNotAvailable(t *testing.T) {
	cfg := &config.Config{
		Azure: &config.CloudProvider{Enabled: true},
	}

	provider, err := NewCloudProvider(cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
	assert.Nil(t, provider)
}

func TestNewCloudProvider_GCPEnabled_Success(t *testing.T) {
	cfg := &config.Config{
		GCP: &config.GCPCloudProvider{
			CloudProvider: config.CloudProvider{Enabled: true},
		},
	}

	provider, err := NewCloudProvider(cfg)

	assert.NoError(t, err)
	assert.IsType(t, &gcp.GCPProvider{}, provider)
}

func TestNewCloudProvider_NoneEnabled_Err(t *testing.T) {
	cfg := &config.Config{}

	provider, err := NewCloudProvider(cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no cloud provider enabled")
	assert.Nil(t, provider)
}
