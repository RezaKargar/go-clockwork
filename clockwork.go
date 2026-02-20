package clockwork

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Clockwork is the runtime service for collecting and serving request metadata.
type Clockwork struct {
	config  Config
	storage Storage

	activeByTrace sync.Map // map[traceID]*Collector
	activeCount   atomic.Int64
}

// NewClockwork creates a new Clockwork service.
func NewClockwork(cfg Config, storage Storage) *Clockwork {
	cfg.Normalize()
	return &Clockwork{
		config:  cfg,
		storage: storage,
	}
}

// Config returns active Clockwork config.
func (c *Clockwork) Config() Config {
	if c == nil {
		return Config{}
	}
	return c.config
}

// Storage returns active storage implementation.
func (c *Clockwork) Storage() Storage {
	if c == nil {
		return nil
	}
	return c.storage
}

// IsEnabled indicates whether Clockwork collection is enabled.
func (c *Clockwork) IsEnabled() bool {
	return c != nil && c.config.Enabled
}

// SaveMetadata stores request metadata.
func (c *Clockwork) SaveMetadata(ctx context.Context, metadata *Metadata) error {
	if c == nil || !c.config.Enabled || c.storage == nil {
		return nil
	}
	if metadata == nil {
		return nil
	}
	return c.storage.Store(ctx, metadata)
}

// GetMetadata fetches metadata by request id.
func (c *Clockwork) GetMetadata(ctx context.Context, id string) (*Metadata, error) {
	if c == nil || c.storage == nil {
		return nil, fmt.Errorf("clockwork storage is not configured")
	}
	return c.storage.Get(ctx, id)
}

// ListMetadata returns recent metadata entries.
func (c *Clockwork) ListMetadata(ctx context.Context, limit int) ([]*Metadata, error) {
	if c == nil || c.storage == nil {
		return nil, fmt.Errorf("clockwork storage is not configured")
	}
	return c.storage.List(ctx, limit)
}

// Cleanup removes old entries from storage.
func (c *Clockwork) Cleanup(ctx context.Context) error {
	if c == nil || c.storage == nil {
		return nil
	}
	return c.storage.Cleanup(ctx, c.config.RequestRetentionTime)
}

// StartCleanupLoop periodically calls Cleanup until stop channel is closed.
func (c *Clockwork) StartCleanupLoop(stop <-chan struct{}) {
	if c == nil {
		return
	}
	c.runCleanup(stop)
}

// NewCollector creates a bounded collector for one request.
func (c *Clockwork) NewCollector(method, uri string) *Collector {
	if c == nil {
		return nil
	}
	return NewCollector(method, uri, limitsFromConfig(c.config))
}

// CompleteRequest finalizes and stores collected request data.
func (c *Clockwork) CompleteRequest(ctx context.Context, collector *Collector, status int, duration time.Duration) error {
	if c == nil || collector == nil {
		return nil
	}

	collector.SetResponseData(status, duration)
	metadata := collector.GetMetadata()
	if metadata == nil {
		return nil
	}

	if metadata.TraceID != "" {
		c.unregisterTrace(metadata.TraceID)
	}

	return c.SaveMetadata(ctx, metadata)
}

// RegisterTrace associates a trace id with the active request collector.
func (c *Clockwork) RegisterTrace(traceID string, collector *Collector) {
	if c == nil || traceID == "" || collector == nil {
		return
	}

	_, loaded := c.activeByTrace.LoadOrStore(traceID, collector)
	if !loaded {
		c.activeCount.Add(1)
	}
}

func (c *Clockwork) unregisterTrace(traceID string) {
	if c == nil || traceID == "" {
		return
	}
	if _, loaded := c.activeByTrace.LoadAndDelete(traceID); loaded {
		c.activeCount.Add(-1)
	}
}

// HasActiveTraces reports whether any request currently has active Clockwork capture.
func (c *Clockwork) HasActiveTraces() bool {
	if c == nil {
		return false
	}
	return c.activeCount.Load() > 0
}

// RecordLogForTrace appends a log entry for a traced active request.
func (c *Clockwork) RecordLogForTrace(traceID, level, message string, fields map[string]interface{}) {
	c.RecordLogForTraceWithTrace(traceID, level, message, fields, nil)
}

// RecordLogForTraceWithTrace appends a log entry for a traced active request with trace frames.
func (c *Clockwork) RecordLogForTraceWithTrace(traceID, level, message string, fields map[string]interface{}, trace []LogTraceFrame) {
	if c == nil || traceID == "" {
		return
	}
	collectorAny, ok := c.activeByTrace.Load(traceID)
	if !ok {
		return
	}
	collector, _ := collectorAny.(*Collector)
	if collector == nil {
		return
	}
	collector.AddLogEntryWithTrace(level, message, fields, trace)
}

// RecordLogForSingleActive appends a log entry when exactly one traced request is active.
// This is a best-effort fallback for log lines that don't carry a trace_id field.
func (c *Clockwork) RecordLogForSingleActive(level, message string, fields map[string]interface{}) bool {
	return c.RecordLogForSingleActiveWithTrace(level, message, fields, nil)
}

// RecordLogForSingleActiveWithTrace appends a log entry with trace frames when exactly one request is active.
func (c *Clockwork) RecordLogForSingleActiveWithTrace(level, message string, fields map[string]interface{}, trace []LogTraceFrame) bool {
	if c == nil || c.activeCount.Load() != 1 {
		return false
	}

	var collector *Collector
	c.activeByTrace.Range(func(_, value interface{}) bool {
		collector, _ = value.(*Collector)
		return false
	})
	if collector == nil {
		return false
	}

	collector.AddLogEntryWithTrace(level, message, fields, trace)
	return true
}

func (c *Clockwork) runCleanup(stop <-chan struct{}) {
	if c == nil || c.config.CleanupInterval <= 0 {
		return
	}
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = c.Cleanup(context.Background())
		case <-stop:
			return
		}
	}
}
