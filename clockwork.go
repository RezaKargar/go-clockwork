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

	dataSources   []DataSource
	dataSourcesMu sync.RWMutex

	activeByTrace sync.Map // map[traceID]*Collector
	activeCount   atomic.Int64

	goroutineMu   sync.Mutex
	goroutineRegs map[uint64]*goroutineReg
}

// goroutineReg reference-counts a goroutine's registration so nested spans
// running on the same goroutine (e.g. a worker goroutine that starts a
// top-level span and then a child DB/cache span within it) don't have an
// inner span's OnEnd tear down the association while an outer span on that
// same goroutine is still in flight.
type goroutineReg struct {
	collector *Collector
	depth     int
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

// RegisterDataSource adds a data source that will be invoked when each request completes.
// Data sources can call collector methods (e.g. SetUserData) to attach custom data.
func (c *Clockwork) RegisterDataSource(ds DataSource) {
	if c == nil || ds == nil {
		return
	}
	c.dataSourcesMu.Lock()
	defer c.dataSourcesMu.Unlock()
	c.dataSources = append(c.dataSources, ds)
}

// CompleteRequest finalizes and stores collected request data.
func (c *Clockwork) CompleteRequest(ctx context.Context, collector *Collector, status int, duration time.Duration) error {
	if c == nil || collector == nil {
		return nil
	}

	collector.SetResponseData(status, duration)

	// TODO: Why we have lock here?!
	c.dataSourcesMu.RLock()
	sources := c.dataSources
	c.dataSourcesMu.RUnlock()
	for _, ds := range sources {
		ds.Resolve(ctx, collector)
	}

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

// RegisterGoroutine associates the calling goroutine with the active request
// collector, so context-free call sites (e.g. a wrapped zapcore.Core, which
// never sees a context.Context) can still be correlated with the right
// collector. Callers must pair every call with a matching UnregisterGoroutine
// for the same goroutine ID; calls nest correctly (a goroutine that's already
// registered just bumps a ref count), so it's safe to call this from every
// span's OnStart on top of the top-level per-request registration.
func (c *Clockwork) RegisterGoroutine(goroutineID uint64, collector *Collector) {
	if c == nil || collector == nil {
		return
	}
	c.goroutineMu.Lock()
	defer c.goroutineMu.Unlock()
	if c.goroutineRegs == nil {
		c.goroutineRegs = make(map[uint64]*goroutineReg)
	}
	if reg, ok := c.goroutineRegs[goroutineID]; ok {
		reg.depth++
		return
	}
	c.goroutineRegs[goroutineID] = &goroutineReg{collector: collector, depth: 1}
}

// UnregisterGoroutine removes one nested registration set by RegisterGoroutine
// for the given goroutine ID, only clearing the association once the
// outermost registration is unregistered.
func (c *Clockwork) UnregisterGoroutine(goroutineID uint64) {
	if c == nil {
		return
	}
	c.goroutineMu.Lock()
	defer c.goroutineMu.Unlock()
	reg, ok := c.goroutineRegs[goroutineID]
	if !ok {
		return
	}
	reg.depth--
	if reg.depth <= 0 {
		delete(c.goroutineRegs, goroutineID)
	}
}

// CollectorForGoroutine returns the collector associated with the given
// goroutine ID via RegisterGoroutine, or nil if none is active.
func (c *Clockwork) CollectorForGoroutine(goroutineID uint64) *Collector {
	if c == nil {
		return nil
	}
	c.goroutineMu.Lock()
	defer c.goroutineMu.Unlock()
	reg, ok := c.goroutineRegs[goroutineID]
	if !ok {
		return nil
	}
	return reg.collector
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

// RecordLogForGoroutine appends a log entry for the collector registered
// against goroutineID via RegisterGoroutine. This is the primary fallback for
// log lines that don't carry a trace_id field: unlike RecordLogForSingleActive,
// it stays correct with any number of concurrently active traced requests, as
// long as the log call happens on the same goroutine that is running the
// traced request handler.
func (c *Clockwork) RecordLogForGoroutine(goroutineID uint64, level, message string, fields map[string]interface{}, trace []LogTraceFrame) bool {
	collector := c.CollectorForGoroutine(goroutineID)
	if collector == nil {
		return false
	}
	collector.AddLogEntryWithTrace(level, message, fields, trace)
	return true
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
