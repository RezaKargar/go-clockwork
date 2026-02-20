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
  storage_type: "memory"
  max_requests: 100
`), 0o600))
	require.NoError(t, os.WriteFile(envPath, []byte("CLOCKWORK_A_STORAGE_TYPE=redis\nCLOCKWORK_A_MAX_REQUESTS=321\n"), 0o600))

	cfg, err := Load(LoadOptions{
		ConfigPath: dir,
		ConfigName: "custom",
		ConfigType: "yml",
		EnvPrefix:  "CLOCKWORK_A",
		EnvFiles:   []string{envPath},
	})
	require.NoError(t, err)
	require.Equal(t, "redis", cfg.StorageType)
	require.Equal(t, 321, cfg.MaxRequests)
}

func TestLoad_FromRootConfigShape(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "clockwork.yml")

	require.NoError(t, os.WriteFile(configPath, []byte(`enabled: true
header_name: "X-Clockwork"
storage_type: "memcache"
memcache_endpoints:
  - "127.0.0.1:11211"
`), 0o600))

	cfg, err := Load(LoadOptions{
		ConfigPath: dir,
		ConfigName: "clockwork",
		ConfigType: "yml",
		EnvPrefix:  "CLOCKWORK_B",
	})
	require.NoError(t, err)
	require.Equal(t, "memcache", cfg.StorageType)
	require.Equal(t, []string{"127.0.0.1:11211"}, cfg.MemcacheEndpoints)
}
