package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hexiosec/asm-cloud-connector/internal/logger"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v3"
)

type CloudProvider struct {
	Enabled bool `yaml:"enabled"`
}

type AWSServices struct {
	CheckEC2          bool `yaml:"check_ec2"`
	CheckEIP          bool `yaml:"check_eip"`
	CheckELB          bool `yaml:"check_elb"`
	CheckS3           bool `yaml:"check_s3"`
	CheckACM          bool `yaml:"check_acm"`
	CheckRoute53      bool `yaml:"check_route53"`
	CheckCloudFront   bool `yaml:"check_cloudfront"`
	CheckAPIGateway   bool `yaml:"check_api_gateway"`
	CheckAPIGatewayV2 bool `yaml:"check_api_gateway_v2"`
	CheckEKS          bool `yaml:"check_eks"`
	CheckRDS          bool `yaml:"check_rds"`
	CheckOpenSearch   bool `yaml:"check_opensearch"`
	CheckLambda       bool `yaml:"check_lambda"`
}

type AWSCloudProvider struct {
	CloudProvider   `yaml:",inline"`
	ListAllAccounts bool         `yaml:"list_all_accounts"`
	Accounts        []string     `yaml:"accounts,omitempty"`
	AssumeRole      *string      `yaml:"assume_role,omitempty" validate:"required_with=Accounts ListAllAccounts"`
	Services        *AWSServices `yaml:"services,omitempty" validate:"required_with=Enabled"`
	APIKeySecret    *string      `yaml:"api_key_secret,omitempty"`
	DefaultRegion   string       `yaml:"default_region" validate:"required"`
}

type Config struct {
	ScanID           string            `yaml:"scan_id" env:"SCAN_ID,overwrite" validate:"required"`
	SeedTag          string            `yaml:"seed_tag" env:"SEED_TAG,overwrite" validate:"required"`
	DeleteStaleSeeds bool              `yaml:"delete_stale_seeds" env:"DELETE_STALE_SEEDS,overwrite"`
	AWS              *AWSCloudProvider `yaml:"aws,omitempty" env:",noinit" validate:"required_without_all=Azure GCP"`
	Azure            *CloudProvider    `yaml:"azure,omitempty" env:",noinit" validate:"required_without_all=AWS GCP"`
	GCP              *CloudProvider    `yaml:"gcp,omitempty" env:",noinit" validate:"required_without_all=AWS Azure"`

	Http struct {
		RetryCount     int           `yaml:"retry_count"  validate:"required"`
		RetryBaseDelay time.Duration `yaml:"retry_base_delay"  validate:"required"`
		RetryMaxDelay  time.Duration `yaml:"retry_max_delay"  validate:"required"`
	} `yaml:"http" validate:"required"`
}

// Provider for Config
func Provider(filePath string) *Config {
	config, err := loadConfig(filePath)
	if err != nil {
		logger.GetGlobalLogger().Fatal().Err(err).Msg("Config failed to load")
	}
	return config
}

func loadConfig(filePath string) (*Config, error) {
	if raw, ok := os.LookupEnv("CONNECTOR_CONFIG"); ok {
		logger.GetGlobalLogger().Info().Msg("Loading config from CONNECTOR_CONFIG env var")

		if strings.TrimSpace(raw) == "" {
			return nil, fmt.Errorf("config: CONNECTOR_CONFIG is set but empty")
		}

		config, err := unmarshalConfig([]byte(raw))
		if err != nil {
			return nil, fmt.Errorf("config: failed to parse CONNECTOR_CONFIG as YAML: %w", err)
		}
		setDefaults(config)
		if err := validate(config); err != nil {
			return nil, fmt.Errorf("config: validation failed for CONNECTOR_CONFIG: %w", err)
		}

		return config, nil
	}

	logger.GetGlobalLogger().Info().Str("path", filePath).Msg("Loading config from file")

	cfgFile, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config: no configuration found: CONNECTOR_CONFIG not set and file %s not found", filePath)
		}
		return nil, fmt.Errorf("config: failed to read %s: %w", filePath, err)
	}

	config, err := unmarshalConfig(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("config: failed to unmarshal %s: %w", filePath, err)
	}

	setDefaults(config)

	if err := validate(config); err != nil {
		return nil, fmt.Errorf("config: validation failed for %s: %w", filePath, err)
	}

	return config, nil
}

func unmarshalConfig(configYaml []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(configYaml, &config); err != nil {
		return nil, err
	}
	if err := envconfig.Process(context.Background(), &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func setDefaults(config *Config) {
	// Apply defaults
	if config.Http.RetryCount == 0 {
		config.Http.RetryCount = 4
	}
	if config.Http.RetryBaseDelay == 0 {
		config.Http.RetryBaseDelay = 1 * time.Second
	}
	if config.Http.RetryMaxDelay == 0 {
		config.Http.RetryMaxDelay = 5 * time.Second
	}
	if config.SeedTag == "" {
		config.SeedTag = "cloud-connector"
	}
}

func validate(config *Config) error {
	return validator.New().Struct(config)
}
