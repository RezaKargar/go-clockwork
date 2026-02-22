package clockwork

import (
	"time"
)

// Config holds Clockwork configuration.
type Config struct {
	Enabled    bool   `mapstructure:"enabled"`
	HeaderName string `mapstructure:"header_name"`
	IDHeader   string `mapstructure:"id_header_name"`

	MaxRequests            int   `mapstructure:"max_requests"`
	MaxStorageBytes        int64 `mapstructure:"max_storage_bytes"`
	MaxRequestPayloadBytes int   `mapstructure:"max_request_payload_bytes"`

	MaxDatabaseQueries int `mapstructure:"max_database_queries"`
	MaxCacheQueries    int `mapstructure:"max_cache_queries"`
	MaxLogEntries      int `mapstructure:"max_log_entries"`
	MaxTimelineEvents  int `mapstructure:"max_timeline_events"`
	MaxStringLength    int `mapstructure:"max_string_length"`

	SlowQueryThreshold   time.Duration `mapstructure:"slow_query_threshold"`
	CleanupInterval      time.Duration `mapstructure:"cleanup_interval"`
	RequestRetentionTime time.Duration `mapstructure:"request_retention_time"`
}

// DefaultConfig returns baseline defaults for new deployments.
func DefaultConfig() Config {
	return Config{
		Enabled:                true,
		HeaderName:             "X-Clockwork",
		IDHeader:               "X-Clockwork-Id",
		MaxRequests:            200,
		MaxStorageBytes:        64 * 1024 * 1024,
		MaxRequestPayloadBytes: 256 * 1024,
		MaxDatabaseQueries:     100,
		MaxCacheQueries:        200,
		MaxLogEntries:          150,
		MaxTimelineEvents:      200,
		MaxStringLength:        2048,
		SlowQueryThreshold:     100 * time.Millisecond,
		CleanupInterval:        5 * time.Minute,
		RequestRetentionTime:   time.Hour,
	}
}

// Normalize applies defaults.
func (c *Config) Normalize() {
	if c == nil {
		return
	}
	d := DefaultConfig()

	if c.HeaderName == "" {
		c.HeaderName = d.HeaderName
	}
	if c.IDHeader == "" {
		c.IDHeader = d.IDHeader
	}
	if c.MaxRequests <= 0 {
		c.MaxRequests = d.MaxRequests
	}
	if c.MaxStorageBytes <= 0 {
		c.MaxStorageBytes = d.MaxStorageBytes
	}
	if c.MaxRequestPayloadBytes <= 0 {
		c.MaxRequestPayloadBytes = d.MaxRequestPayloadBytes
	}
	if c.MaxDatabaseQueries <= 0 {
		c.MaxDatabaseQueries = d.MaxDatabaseQueries
	}
	if c.MaxCacheQueries <= 0 {
		c.MaxCacheQueries = d.MaxCacheQueries
	}
	if c.MaxLogEntries <= 0 {
		c.MaxLogEntries = d.MaxLogEntries
	}
	if c.MaxTimelineEvents <= 0 {
		c.MaxTimelineEvents = d.MaxTimelineEvents
	}
	if c.MaxStringLength <= 0 {
		c.MaxStringLength = d.MaxStringLength
	}
	if c.SlowQueryThreshold <= 0 {
		c.SlowQueryThreshold = d.SlowQueryThreshold
	}
	if c.CleanupInterval <= 0 {
		c.CleanupInterval = d.CleanupInterval
	}
	if c.RequestRetentionTime <= 0 {
		c.RequestRetentionTime = d.RequestRetentionTime
	}
}
