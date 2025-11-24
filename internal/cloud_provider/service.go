package cloud_provider

import (
	"fmt"

	"github.com/hexiosec/asm-cloud-connector/internal/aws"
	t "github.com/hexiosec/asm-cloud-connector/internal/cloud_provider/types"
	"github.com/hexiosec/asm-cloud-connector/internal/config"
)

func NewCloudProvider(cfg *config.Config) (t.CloudProvider, error) {
	switch {
	case cfg.AWS != nil && cfg.AWS.Enabled:
		return aws.NewAWSProvider(cfg)
	case cfg.Azure != nil && cfg.Azure.Enabled:
		return nil, fmt.Errorf("cloud provider Azure not available")
	case cfg.GCP != nil && cfg.GCP.Enabled:
		return nil, fmt.Errorf("cloud provider GCP not available")
	default:
		return nil, fmt.Errorf("no cloud provider enabled")
	}
}
