package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_IgnoreList(t *testing.T) {
	const base = "scan_id: scan-123\naws:\n  enabled: true\n  default_region: eu-west-2\n  services:\n    check_route53: true\n"
	tests := []struct {
		name     string
		yaml     string
		fromEnv  bool
		override string
		want     []string
	}{
		{name: "omitted"},
		{name: "empty", yaml: "ignore_list: []\n", want: []string{}},
		{name: "file", yaml: "ignore_list:\n  - third-party.example.com\n  - 203.0.113.10\n", want: []string{"third-party.example.com", "203.0.113.10"}},
		{name: "CONNECTOR_CONFIG", fromEnv: true, yaml: "ignore_list: [third-party.example.com]\n", want: []string{"third-party.example.com"}},
		{name: "environment overrides file", yaml: "ignore_list: [from-file.example.com]\n", override: "from-env.example.com,203.0.113.10", want: []string{"from-env.example.com", "203.0.113.10"}},
		{name: "environment overrides CONNECTOR_CONFIG", fromEnv: true, yaml: "ignore_list: [from-config.example.com]\n", override: "from-env.example.com", want: []string{"from-env.example.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("IGNORE_LIST", tt.override)
			path := filepath.Join(t.TempDir(), "config.yml")
			if tt.fromEnv {
				t.Setenv("CONNECTOR_CONFIG", base+tt.yaml)
			} else {
				// An empty CONNECTOR_CONFIG is an error, so ensure it is unset.
				t.Setenv("CONNECTOR_CONFIG", "")
				require.NoError(t, os.Unsetenv("CONNECTOR_CONFIG"))
				require.NoError(t, os.WriteFile(path, []byte(base+tt.yaml), 0600))
			}
			cfg, err := loadConfig(path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, cfg.IgnoreList)
		})
	}
}
