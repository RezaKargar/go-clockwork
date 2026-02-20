package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/RezaKargar/go-clockwork"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

// LoadOptions controls how config is loaded from yml and env.
type LoadOptions struct {
	ConfigPath string
	ConfigName string
	ConfigType string
	EnvPrefix  string
	EnvFiles   []string
}

// Load reads configuration from yml and .env files then applies env overrides.
func Load(opts LoadOptions) (clockwork.Config, error) {
	cfg := clockwork.DefaultConfig()

	if err := loadDotEnv(opts.EnvFiles); err != nil {
		return clockwork.Config{}, err
	}

	v := viper.New()
	configPath := strings.TrimSpace(opts.ConfigPath)
	if configPath == "" {
		configPath = "."
	}
	configName := strings.TrimSpace(opts.ConfigName)
	if configName == "" {
		configName = "clockwork"
	}
	configType := strings.TrimSpace(opts.ConfigType)
	if configType == "" {
		configType = "yml"
	}

	v.SetConfigName(configName)
	v.SetConfigType(configType)
	v.AddConfigPath(configPath)

	envPrefix := strings.TrimSpace(opts.EnvPrefix)
	if envPrefix == "" {
		envPrefix = "CLOCKWORK"
	}
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	bindClockworkEnv(v, envPrefix)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return clockwork.Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	if v.IsSet("clockwork") {
		if err := v.UnmarshalKey("clockwork", &cfg); err != nil {
			return clockwork.Config{}, fmt.Errorf("unmarshal clockwork section: %w", err)
		}
	} else {
		if err := v.Unmarshal(&cfg); err != nil {
			return clockwork.Config{}, fmt.Errorf("unmarshal config root: %w", err)
		}
	}
	applyEnvOverrides(&cfg, envPrefix)

	cfg.Normalize()
	return cfg, nil
}

func bindClockworkEnv(v *viper.Viper, envPrefix string) {
	if v == nil {
		return
	}
	keys := map[string]string{
		"enabled":                   "ENABLED",
		"header_name":               "HEADER_NAME",
		"id_header_name":            "ID_HEADER_NAME",
		"storage_type":              "STORAGE_TYPE",
		"max_requests":              "MAX_REQUESTS",
		"max_storage_bytes":         "MAX_STORAGE_BYTES",
		"max_request_payload_bytes": "MAX_REQUEST_PAYLOAD_BYTES",
		"max_database_queries":      "MAX_DATABASE_QUERIES",
		"max_cache_queries":         "MAX_CACHE_QUERIES",
		"max_log_entries":           "MAX_LOG_ENTRIES",
		"max_timeline_events":       "MAX_TIMELINE_EVENTS",
		"max_string_length":         "MAX_STRING_LENGTH",
		"slow_query_threshold":      "SLOW_QUERY_THRESHOLD",
		"cleanup_interval":          "CLEANUP_INTERVAL",
		"request_retention_time":    "REQUEST_RETENTION_TIME",
		"redis_endpoint":            "REDIS_ENDPOINT",
		"redis_password":            "REDIS_PASSWORD",
		"redis_db":                  "REDIS_DB",
		"redis_prefix":              "REDIS_PREFIX",
		"memcache_endpoints":        "MEMCACHE_ENDPOINTS",
		"memcache_prefix":           "MEMCACHE_PREFIX",
	}

	for key, suffix := range keys {
		envKey := envPrefix + "_" + suffix
		_ = v.BindEnv(key, envKey)
		_ = v.BindEnv("clockwork."+key, envKey)
	}
}

func applyEnvOverrides(cfg *clockwork.Config, envPrefix string) {
	if cfg == nil || strings.TrimSpace(envPrefix) == "" {
		return
	}
	key := func(name string) string { return envPrefix + "_" + name }

	if value, ok := lookupEnv(key("ENABLED")); ok {
		if parsed, err := strconv.ParseBool(value); err == nil {
			cfg.Enabled = parsed
		}
	}
	if value, ok := lookupEnv(key("HEADER_NAME")); ok {
		cfg.HeaderName = value
	}
	if value, ok := lookupEnv(key("ID_HEADER_NAME")); ok {
		cfg.IDHeader = value
	}
	if value, ok := lookupEnv(key("STORAGE_TYPE")); ok {
		cfg.StorageType = value
	}
	if value, ok := lookupEnv(key("MAX_REQUESTS")); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.MaxRequests = parsed
		}
	}
	if value, ok := lookupEnv(key("MAX_STORAGE_BYTES")); ok {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			cfg.MaxStorageBytes = parsed
		}
	}
	if value, ok := lookupEnv(key("MAX_REQUEST_PAYLOAD_BYTES")); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.MaxRequestPayloadBytes = parsed
		}
	}
	if value, ok := lookupEnv(key("MAX_DATABASE_QUERIES")); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.MaxDatabaseQueries = parsed
		}
	}
	if value, ok := lookupEnv(key("MAX_CACHE_QUERIES")); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.MaxCacheQueries = parsed
		}
	}
	if value, ok := lookupEnv(key("MAX_LOG_ENTRIES")); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.MaxLogEntries = parsed
		}
	}
	if value, ok := lookupEnv(key("MAX_TIMELINE_EVENTS")); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.MaxTimelineEvents = parsed
		}
	}
	if value, ok := lookupEnv(key("MAX_STRING_LENGTH")); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.MaxStringLength = parsed
		}
	}
	if value, ok := lookupEnv(key("SLOW_QUERY_THRESHOLD")); ok {
		if parsed, err := time.ParseDuration(value); err == nil {
			cfg.SlowQueryThreshold = parsed
		}
	}
	if value, ok := lookupEnv(key("CLEANUP_INTERVAL")); ok {
		if parsed, err := time.ParseDuration(value); err == nil {
			cfg.CleanupInterval = parsed
		}
	}
	if value, ok := lookupEnv(key("REQUEST_RETENTION_TIME")); ok {
		if parsed, err := time.ParseDuration(value); err == nil {
			cfg.RequestRetentionTime = parsed
		}
	}
	if value, ok := lookupEnv(key("REDIS_ENDPOINT")); ok {
		cfg.RedisEndpoint = value
	}
	if value, ok := lookupEnv(key("REDIS_PASSWORD")); ok {
		cfg.RedisPassword = value
	}
	if value, ok := lookupEnv(key("REDIS_DB")); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.RedisDB = parsed
		}
	}
	if value, ok := lookupEnv(key("REDIS_PREFIX")); ok {
		cfg.RedisPrefix = value
	}
	if value, ok := lookupEnv(key("MEMCACHE_ENDPOINTS")); ok {
		parts := strings.Split(value, ",")
		endpoints := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				endpoints = append(endpoints, trimmed)
			}
		}
		cfg.MemcacheEndpoints = endpoints
	}
	if value, ok := lookupEnv(key("MEMCACHE_PREFIX")); ok {
		cfg.MemcachePrefix = value
	}
}

func lookupEnv(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func loadDotEnv(envFiles []string) error {
	if len(envFiles) == 0 {
		return nil
	}

	resolved := make([]string, 0, len(envFiles))
	for _, file := range envFiles {
		trimmed := strings.TrimSpace(file)
		if trimmed == "" {
			continue
		}
		if _, err := os.Stat(trimmed); err == nil {
			resolved = append(resolved, trimmed)
			continue
		}
		if filepath.IsAbs(trimmed) {
			continue
		}
	}

	if len(resolved) == 0 {
		return nil
	}
	if err := gotenv.Load(resolved...); err != nil {
		return fmt.Errorf("load env files: %w", err)
	}
	return nil
}
