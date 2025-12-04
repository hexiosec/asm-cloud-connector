package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DefaultConfigParsing_Success(t *testing.T) {
	testFile := []byte(strings.ReplaceAll(`
		scan_id: 00000000-0000-0000-0000-000000000000
		seed_tag: cloud_connector
		aws:
			enabled: false
			default_region: region
		azure:
			enabled: false
		gcp:
			enabled: false
	`, "\t", "  "))

	// Create test config file
	cfgFilePath := t.TempDir() + "/config.yml"
	err := os.WriteFile(cfgFilePath, testFile, 0777)
	assert.NoError(t, err, "Failed to write test config file")

	config := Provider(cfgFilePath)
	assert.NoError(t, err)

	assert.Equal(t, "00000000-0000-0000-0000-000000000000", config.ScanID)
	assert.Equal(t, "cloud_connector", config.SeedTag)
	assert.False(t, config.DeleteStaleSeeds)
	assert.False(t, config.AWS.Enabled)
	assert.False(t, config.Azure.Enabled)
	assert.False(t, config.GCP.Enabled)
	assert.Equal(t, 4, config.Http.RetryCount)                 // Default value
	assert.Equal(t, 1*time.Second, config.Http.RetryBaseDelay) // Default value
	assert.Equal(t, 5*time.Second, config.Http.RetryMaxDelay)  // Default value
	assert.Nil(t, config.AWS.AssumeRole)
}

func Test_NoCloudProviders_Fails(t *testing.T) {
	testFile := []byte(strings.ReplaceAll(`
		scan_id: 00000000-0000-0000-0000-000000000000
		seed_tag: cloud_connector
	`, "\t", "  "))
	// Parsing the file passes
	config, err := unmarshalConfig(testFile)
	assert.NoError(t, err)

	// Validating the file fails
	err = validate(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed on the 'required_without_all' tag")
}

func Test_OverrideHttpDefault_Success(t *testing.T) {
	testFile := []byte(strings.ReplaceAll(`
		scan_id: 00000000-0000-0000-0000-000000000000
		seed_tag: cloud_connector
		aws:
			enabled: false
			default_region: region
		azure:
			enabled: false
		gcp:
			enabled: false
		http:
			retry_count: 10
	`, "\t", "  "))
	// Create test config file
	cfgFilePath := t.TempDir() + "/config.yml"
	err := os.WriteFile(cfgFilePath, testFile, 0777)
	assert.NoError(t, err, "Failed to write test config file")

	config := Provider(cfgFilePath)
	assert.NoError(t, err)

	assert.Equal(t, "00000000-0000-0000-0000-000000000000", config.ScanID)
	assert.Equal(t, "cloud_connector", config.SeedTag)
	assert.False(t, config.AWS.Enabled)
	assert.False(t, config.Azure.Enabled)
	assert.False(t, config.GCP.Enabled)
	assert.Equal(t, 10, config.Http.RetryCount)
	assert.Equal(t, 1*time.Second, config.Http.RetryBaseDelay) // Default value
	assert.Equal(t, 5*time.Second, config.Http.RetryMaxDelay)  // Default value
}

func Test_AWSAssumeRoleRequired(t *testing.T) {
	tests := []struct {
		name     string
		testFile string
		success  bool
	}{
		{
			name: "listAllAccounts_NoAssumeRoleInConfig_Fail",
			testFile: `
				scan_id: 00000000-0000-0000-0000-000000000000
				seed_tag: cloud_connector
				aws:
					enabled: true
					list_all_accounts: true
					default_region: region
					services:
						check_ec2: true
						check_eip: true
						check_elb: true
						check_s3: true
						check_acm: true
						check_route53: true
						check_cloudfront: true
						check_api_gateway: true
						check_api_gateway_v2: true
						check_eks: true
						check_rds: true
						check_opensearch: true
						check_lambda: true
				http:
					retry_count: 4
					retry_base_delay: 1s
					retry_max_delay: 5m
			`,
			success: false,
		},
		{
			name: "listAllAccounts_AssumeRoleInConfig_Success",
			testFile: `
				scan_id: 00000000-0000-0000-0000-000000000000
				seed_tag: cloud_connector
				aws:
					enabled: true
					list_all_accounts: true
					assume_role: role
					default_region: region
					services:
						check_ec2: true
						check_eip: true
						check_elb: true
						check_s3: true
						check_acm: true
						check_route53: true
						check_cloudfront: true
						check_api_gateway: true
						check_api_gateway_v2: true
						check_eks: true
						check_rds: true
						check_opensearch: true
						check_lambda: true
				http:
					retry_count: 4
					retry_base_delay: 1s
					retry_max_delay: 5m
			`,
			success: true,
		},
		{
			name: "accounts_NoAssumeRoleInConfig_Fail",
			testFile: `
				scan_id: 00000000-0000-0000-0000-000000000000
				seed_tag: cloud_connector
				aws:
					enabled: true
					accounts: [123456]
					default_region: region
					services:
						check_ec2: true
						check_eip: true
						check_elb: true
						check_s3: true
						check_acm: true
						check_route53: true
						check_cloudfront: true
						check_api_gateway: true
						check_api_gateway_v2: true
						check_eks: true
						check_rds: true
						check_opensearch: true
						check_lambda: true
				http:
					retry_count: 4
					retry_base_delay: 1s
					retry_max_delay: 5m
			`,
			success: false,
		},
		{
			name: "accounts_AssumeRoleInConfig_Success",
			testFile: `
				scan_id: 00000000-0000-0000-0000-000000000000
				seed_tag: cloud_connector
				aws:
					enabled: true
					accounts: [123456]
					assume_role: role
					default_region: region
					services:
						check_ec2: true
						check_eip: true
						check_elb: true
						check_s3: true
						check_acm: true
						check_route53: true
						check_cloudfront: true
						check_api_gateway: true
						check_api_gateway_v2: true
						check_eks: true
						check_rds: true
						check_opensearch: true
						check_lambda: true
				http:
					retry_count: 4
					retry_base_delay: 1s
					retry_max_delay: 5m
			`,
			success: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Parsing the file passes
			config, err := unmarshalConfig([]byte(strings.ReplaceAll(tc.testFile, "\t", "  ")))
			assert.NoError(t, err)

			assert.Equal(t, "00000000-0000-0000-0000-000000000000", config.ScanID)
			assert.Equal(t, "cloud_connector", config.SeedTag)
			assert.True(t, config.AWS.Enabled)

			// Validate the file
			err = validate(config)
			if tc.success {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "'AssumeRole' failed on the 'required_with' tag")
			}
		})
	}
}

func Test_LoadFromEnvConfig_Success(t *testing.T) {
	configYAML := strings.ReplaceAll(`
		scan_id: 00000000-0000-0000-0000-000000000000
		seed_tag: cloud_connector
		delete_stale_seeds: true
		aws:
			enabled: true
			default_region: region
			services:
				check_ec2: true
	`, "\t", "  ")
	t.Setenv("CONNECTOR_CONFIG", configYAML)

	config, err := loadConfig("unused.yml")
	require.NoError(t, err)

	assert.Equal(t, "00000000-0000-0000-0000-000000000000", config.ScanID)
	assert.Equal(t, "cloud_connector", config.SeedTag)
	assert.True(t, config.DeleteStaleSeeds)
	assert.True(t, config.AWS.Enabled)
}

func Test_LoadFromEnvConfig_InvalidYAML(t *testing.T) {
	t.Setenv("CONNECTOR_CONFIG", ":%not-yaml%")

	_, err := loadConfig("unused.yml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CONNECTOR_CONFIG")
	assert.Contains(t, err.Error(), "failed to parse")
}

func Test_LoadFromFile_FallbackWhenEnvUnset(t *testing.T) {
	_ = os.Unsetenv("CONNECTOR_CONFIG")

	testFile := []byte(strings.ReplaceAll(`
		scan_id: 00000000-0000-0000-0000-000000000000
		seed_tag: cloud_connector
		aws:
			enabled: true
			default_region: region
			services:
				check_ec2: true
	`, "\t", "  "))
	cfgFilePath := t.TempDir() + "/config.yml"
	err := os.WriteFile(cfgFilePath, testFile, 0777)
	require.NoError(t, err, "Failed to write test config file")

	config, err := loadConfig(cfgFilePath)
	require.NoError(t, err)

	assert.Equal(t, "00000000-0000-0000-0000-000000000000", config.ScanID)
	assert.Equal(t, "cloud_connector", config.SeedTag)
	assert.True(t, config.AWS.Enabled)
}

func Test_GCPProjectsValidation(t *testing.T) {
	tests := []struct {
		name        string
		testFile    string
		shouldErr   bool
		errText     string
		enabled     bool
		numProjects int
	}{
		{
			name: "enabled_NoProjects_Fail",
			testFile: `
				scan_id: 00000000-0000-0000-0000-000000000000
				seed_tag: cloud_connector
				gcp:
					enabled: true
					services:
						check_dns_resource_record_set: true
				http:
					retry_count: 4
					retry_base_delay: 1s
					retry_max_delay: 5m
			`,
			shouldErr: true,
			errText:   "'Projects' failed on the 'required_with' tag",
		},
		{
			name: "enabled_EmptyProjects_Fail",
			testFile: `
				scan_id: 00000000-0000-0000-0000-000000000000
				seed_tag: cloud_connector
				gcp:
					enabled: true
					projects: []
					services:
						check_dns_resource_record_set: true
				http:
					retry_count: 4
					retry_base_delay: 1s
					retry_max_delay: 5m
			`,
			shouldErr: true,
			errText:   "'Projects' failed on the 'min' tag",
		},
		{
			name: "enabled_ProjectsProvided_Success",
			testFile: `
				scan_id: 00000000-0000-0000-0000-000000000000
				seed_tag: cloud_connector
				gcp:
					enabled: true
					projects: [projects/123456789]
					services:
						check_dns_resource_record_set: true
				http:
					retry_count: 4
					retry_base_delay: 1s
					retry_max_delay: 5m
			`,
			shouldErr:   false,
			enabled:     true,
			numProjects: 1,
		},
		{
			name: "enabled_BadFormatProject_Fail",
			testFile: `
				scan_id: 00000000-0000-0000-0000-000000000000
				seed_tag: cloud_connector
				gcp:
					enabled: true
					projects: [projects/PROJECT_ID]
					services:
						check_dns_resource_record_set: true
				http:
					retry_count: 4
					retry_base_delay: 1s
					retry_max_delay: 5m
			`,
			shouldErr: true,
			errText:   "failed on the 'gcp_project' tag",
		},
		{
			name: "disabled_NoProjects_Success",
			testFile: `
				scan_id: 00000000-0000-0000-0000-000000000000
				seed_tag: cloud_connector
				gcp:
					enabled: false
					services:
						check_dns_resource_record_set: true
				azure:
					enabled: true
				http:
					retry_count: 4
					retry_base_delay: 1s
					retry_max_delay: 5m
			`,
			shouldErr:   false,
			enabled:     false,
			numProjects: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config, err := unmarshalConfig([]byte(strings.ReplaceAll(tc.testFile, "\t", "  ")))
			require.NoError(t, err)

			err = validate(config)
			if tc.shouldErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errText)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.enabled, config.GCP.Enabled)
			assert.Len(t, config.GCP.Projects, tc.numProjects)
		})
	}
}
