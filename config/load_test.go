package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad_FromYAMLAndDotEnvOverride(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom.yml")
	envPath := filepath.Join(dir, ".env")

	require.NoError(t, os.WriteFile(configPath, []byte(`clockwork:
  enabled: true
  header_name: "X-Clockwork"
  max_requests: 100
`), 0o600))
	require.NoError(t, os.WriteFile(envPath, []byte("CLOCKWORK_A_MAX_REQUESTS=321\n"), 0o600))

	cfg, err := Load(LoadOptions{
		ConfigPath: dir,
		ConfigName: "custom",
		ConfigType: "yml",
		EnvPrefix:  "CLOCKWORK_A",
		EnvFiles:   []string{envPath},
	})
	require.NoError(t, err)
	require.Equal(t, 321, cfg.MaxRequests)
	require.True(t, cfg.Enabled)
	require.Equal(t, "X-Clockwork", cfg.HeaderName)
}

func TestLoad_FromRootConfigShape(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "clockwork.yml")

	require.NoError(t, os.WriteFile(configPath, []byte(`enabled: true
header_name: "X-Clockwork"
max_requests: 50
`), 0o600))

	cfg, err := Load(LoadOptions{
		ConfigPath: dir,
		ConfigName: "clockwork",
		ConfigType: "yml",
		EnvPrefix:  "CLOCKWORK_B",
	})
	require.NoError(t, err)
	require.True(t, cfg.Enabled)
	require.Equal(t, "X-Clockwork", cfg.HeaderName)
	require.Equal(t, 50, cfg.MaxRequests)
}
