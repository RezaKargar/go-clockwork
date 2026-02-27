package clockwork

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	maxLogTraceFrames     = 12
	maxLogTraceStackDepth = 32
)

type collectorLimits struct {
	maxRequestBytes  int
	maxStringLen     int
	maxDBQueries     int
	maxCacheQueries  int
	maxLogs          int
	maxTimelineEvent int
}

func limitsFromConfig(cfg Config) collectorLimits {
	return collectorLimits{
		maxRequestBytes:  cfg.MaxRequestPayloadBytes,
		maxStringLen:     cfg.MaxStringLength,
		maxDBQueries:     cfg.MaxDatabaseQueries,
		maxCacheQueries:  cfg.MaxCacheQueries,
		maxLogs:          cfg.MaxLogEntries,
		maxTimelineEvent: cfg.MaxTimelineEvents,
	}
}

// Logger is a minimal logger used by middleware and adapters.
// Implementations can wrap zap, slog, or any logger.
type Logger interface {
	Warn(msg string, keysAndValues ...interface{})
}

// DataCollector is the interface for per-request data collection.
// *Collector implements this interface; custom collectors can implement it to work with Clockwork.
type DataCollector interface {
	ID() string
	SetHeaders(headers map[string]string)
	SetURL(url string)
	SetController(controller string)
	SetTrace(traceID, spanID string)
	SetResponseData(status int, duration time.Duration)
	AddDatabaseQuery(query string, duration time.Duration, connection string, slow bool)
	AddCacheQuery(cacheType, key string, duration time.Duration)
	AddLogEntry(level, message string, fields map[string]interface{})
	AddLogEntryWithTrace(level, message string, fields map[string]interface{}, trace []LogTraceFrame)
	AddTimelineEvent(name, description string, start, end time.Time, color string)
	SetUserData(key string, value interface{})
	GetMetadata() *Metadata
}

// Collector represents per-request Clockwork data collection.
type Collector struct {
	id               string
	startTime        time.Time
	method           string
	uri              string
	url              string
	controller       string
	headers          map[string]string
	traceID          string
	spanID           string
	responseStatus   int
	responseTime     time.Time
	responseDuration time.Duration
	memoryUsageStart uint64
	memoryUsageEnd   uint64

	databaseQueries []DatabaseQuery
	cacheQueries    []CacheQuery
	logEntries      []LogEntry
	timelineEvents  []TimelineEvent
	userData        map[string]interface{}
	dropped         map[string]int
	truncated       bool

	limits    collectorLimits
	usedBytes int

	mu sync.RWMutex
}

// NewCollector creates a new Collector for a request.
func NewCollector(method, uri string, limits collectorLimits) *Collector {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	if limits.maxStringLen <= 0 {
		limits.maxStringLen = 2048
	}

	return &Collector{
		id:               uuid.New().String(),
		startTime:        time.Now(),
		method:           method,
		uri:              uri,
		headers:          make(map[string]string),
		databaseQueries:  make([]DatabaseQuery, 0, 8),
		cacheQueries:     make([]CacheQuery, 0, 16),
		logEntries:       make([]LogEntry, 0, 16),
		timelineEvents:   make([]TimelineEvent, 0, 16),
		dropped:          make(map[string]int),
		userData:         make(map[string]interface{}),
		memoryUsageStart: mem.Alloc,
		limits:           limits,
	}
}

// ID returns the collector ID.
func (c *Collector) ID() string {
	if c == nil {
		return ""
	}
	return c.id
}

// SetResponseData sets response metadata.
func (c *Collector) SetResponseData(status int, duration time.Duration) {
	if c == nil {
		return
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.responseStatus = status
	c.responseTime = time.Now()
	c.responseDuration = duration
	c.memoryUsageEnd = mem.Alloc

	start := unixFromTime(c.startTime)
	end := unixFromTime(c.responseTime)
	c.appendTimelineLocked("request", c.method+" "+c.uri, start, end, "green")
}

// SetHeaders sets request headers.
func (c *Collector) SetHeaders(headers map[string]string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers = headers
}

// SetURL sets the request URL.
func (c *Collector) SetURL(url string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.url = c.truncate(url)
}

// SetController sets the request controller/handler identifier.
func (c *Collector) SetController(controller string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.controller = c.truncate(controller)
}

// SetTrace sets trace and span identifiers.
func (c *Collector) SetTrace(traceID, spanID string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.traceID = c.truncate(traceID)
	c.spanID = c.truncate(spanID)
}

// AddDatabaseQuery adds a database query event.
// Model is auto-extracted from the SQL when not provided via AddDatabaseQueryDetailed.
func (c *Collector) AddDatabaseQuery(query string, duration time.Duration, connection string, slow bool) {
	c.AddDatabaseQueryDetailed(query, duration, connection, slow, "", "", 0)
}

// AddDatabaseQueryDetailed adds a database query with explicit model, file, and line info.
// If model is empty, the table name is extracted from the SQL query.
// If file is empty, it is captured from the caller's stack.
func (c *Collector) AddDatabaseQueryDetailed(query string, duration time.Duration, connection string, slow bool, model, file string, line int) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.reserveLocked("database", c.limits.maxDBQueries, len(c.databaseQueries), len(query)+64) {
		return
	}

	if model == "" {
		model = extractTableName(query)
	}
	if file == "" {
		file, line = callerOutsidePackage(4)
	}

	durationMS := durationMs(duration)
	dq := DatabaseQuery{
		Query:      c.truncate(query),
		Duration:   durationMS,
		Connection: c.truncate(connection),
		Model:      c.truncate(model),
		File:       c.truncate(file),
		Line:       line,
		Slow:       slow,
		Timestamp:  unixTimestamp(),
	}
	c.databaseQueries = append(c.databaseQueries, dq)
	c.appendTimelineLocked("db", dq.Query, dq.Timestamp-durationMS, dq.Timestamp, colorForSlow(slow))
}

// AddCacheQuery adds a cache operation event.
func (c *Collector) AddCacheQuery(cacheType, key string, duration time.Duration) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.reserveLocked("cache", c.limits.maxCacheQueries, len(c.cacheQueries), len(key)+48) {
		return
	}

	durationMS := durationMs(duration)
	cq := CacheQuery{
		Type:      c.truncate(cacheType),
		Key:       c.truncate(key),
		Duration:  durationMS,
		Timestamp: unixTimestamp(),
	}
	c.cacheQueries = append(c.cacheQueries, cq)
	c.appendTimelineLocked("cache", cq.Type+": "+cq.Key, cq.Timestamp-durationMS, cq.Timestamp, "purple")
}

// AddLogEntry adds a log message.
func (c *Collector) AddLogEntry(level, message string, fields map[string]interface{}) {
	c.AddLogEntryWithTrace(level, message, fields, nil)
}

// AddLogEntryWithTrace adds a log message with stack trace frames.
func (c *Collector) AddLogEntryWithTrace(level, message string, fields map[string]interface{}, trace []LogTraceFrame) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	traceBytesEstimate := len(trace) * 64
	if !c.reserveLocked("logs", c.limits.maxLogs, len(c.logEntries), len(message)+96+traceBytesEstimate) {
		return
	}

	sanitizedTrace := c.sanitizeTrace(trace)
	if len(sanitizedTrace) == 0 {
		sanitizedTrace = c.captureCurrentStackTrace(4)
	}

	entry := LogEntry{
		Level:     c.truncate(level),
		Message:   c.truncate(message),
		Context:   c.sanitizeContext(fields),
		Timestamp: unixTimestamp(),
		Trace:     sanitizedTrace,
	}
	c.logEntries = append(c.logEntries, entry)
}

// AddTimelineEvent adds a direct timeline event.
func (c *Collector) AddTimelineEvent(name, description string, start, end time.Time, color string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.reserveLocked("timeline", c.limits.maxTimelineEvent, len(c.timelineEvents), len(name)+len(description)+32) {
		return
	}

	c.appendTimelineLocked(name, description, unixFromTime(start), unixFromTime(end), color)
}

// SetUserData attaches a key-value pair for custom data (e.g. from a DataSource).
// Values are included in Metadata.UserData and shown in the Clockwork UI.
func (c *Collector) SetUserData(key string, value interface{}) {
	if c == nil || key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.userData == nil {
		c.userData = make(map[string]interface{})
	}
	c.userData[key] = value
}

// GetMetadata returns collected metadata.
func (c *Collector) GetMetadata() *Metadata {
	if c == nil {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	totalDBDuration := 0.0
	for _, q := range c.databaseQueries {
		totalDBDuration += q.Duration
	}

	memoryDelta := uint64(0)
	if c.memoryUsageEnd > c.memoryUsageStart {
		memoryDelta = c.memoryUsageEnd - c.memoryUsageStart
	}

	meta := &Metadata{
		ID:                   c.id,
		Version:              1,
		Type:                 "request",
		Time:                 unixFromTime(c.startTime),
		ResponseTime:         unixFromTime(c.responseTime),
		ResponseStatus:       c.responseStatus,
		ResponseDuration:     durationMs(c.responseDuration),
		Method:               c.method,
		URI:                  c.uri,
		URL:                  c.url,
		Controller:           c.controller,
		Headers:              c.headers,
		TraceID:              c.traceID,
		SpanID:               c.spanID,
		DatabaseQueries:      copyDB(c.databaseQueries),
		DatabaseQueriesCount: len(c.databaseQueries),
		DatabaseDuration:     totalDBDuration,
		CacheQueries:         copyCache(c.cacheQueries),
		LogEntries:           copyLogs(c.logEntries),
		TimelineEvents:       copyTimeline(c.timelineEvents),
		MemoryUsage:          memoryDelta,
		Truncated:            c.truncated,
	}

	if len(c.userData) > 0 {
		meta.UserData = make(map[string]interface{}, len(c.userData))
		for k, v := range c.userData {
			meta.UserData[k] = v
		}
	}

	if len(c.dropped) > 0 {
		meta.Dropped = make(map[string]int, len(c.dropped))
		for k, v := range c.dropped {
			meta.Dropped[k] = v
		}
	}

	return meta
}

func (c *Collector) reserveLocked(bucket string, max, current, estimate int) bool {
	if max > 0 && current >= max {
		c.dropped[bucket]++
		c.truncated = true
		return false
	}

	if c.limits.maxRequestBytes > 0 {
		if estimate <= 0 {
			estimate = 1
		}
		if c.usedBytes+estimate > c.limits.maxRequestBytes {
			c.dropped["payload"]++
			c.truncated = true
			return false
		}
		c.usedBytes += estimate
	}

	return true
}

func (c *Collector) appendTimelineLocked(name, description string, startMs, endMs float64, color string) {
	if c.limits.maxTimelineEvent > 0 && len(c.timelineEvents) >= c.limits.maxTimelineEvent {
		c.dropped["timeline"]++
		c.truncated = true
		return
	}

	event := TimelineEvent{
		Name:        c.truncate(name),
		Description: c.truncate(description),
		Start:       startMs,
		Color:       color,
	}

	if endMs > startMs {
		event.End = endMs
		event.Duration = endMs - startMs
	}

	c.timelineEvents = append(c.timelineEvents, event)
}

func (c *Collector) sanitizeContext(fields map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}

	out := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		if len(out) >= 20 {
			c.dropped["log_context"]++
			c.truncated = true
			break
		}

		key := c.truncate(k)
		switch tv := v.(type) {
		case string:
			out[key] = c.truncate(tv)
		case int, int32, int64, uint, uint32, uint64, float32, float64, bool:
			out[key] = tv
		default:
			out[key] = c.truncate(toCompactString(tv))
		}
	}

	return out
}

func (c *Collector) sanitizeTrace(trace []LogTraceFrame) []LogTraceFrame {
	if len(trace) == 0 {
		return nil
	}

	maxFrames := maxLogTraceFrames
	if len(trace) < maxFrames {
		maxFrames = len(trace)
	}
	out := make([]LogTraceFrame, 0, maxFrames)
	for i := 0; i < len(trace) && i < maxLogTraceFrames; i++ {
		out = append(out, LogTraceFrame{
			Call:     c.truncate(trace[i].Call),
			File:     c.truncate(trace[i].File),
			Line:     trace[i].Line,
			IsVendor: trace[i].IsVendor,
		})
	}
	return out
}

func (c *Collector) captureCurrentStackTrace(skip int) []LogTraceFrame {
	pcs := make([]uintptr, maxLogTraceStackDepth)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return nil
	}
	frames := runtime.CallersFrames(pcs[:n])
	out := make([]LogTraceFrame, 0, maxLogTraceFrames)
	for len(out) < maxLogTraceFrames {
		frame, more := frames.Next()
		if frame.File == "" {
			if !more {
				break
			}
			continue
		}
		out = append(out, LogTraceFrame{
			Call:     c.truncate(frame.Function),
			File:     c.truncate(frame.File),
			Line:     frame.Line,
			IsVendor: isVendorPath(frame.File),
		})
		if !more {
			break
		}
	}
	return out
}

func (c *Collector) truncate(v string) string {
	if c.limits.maxStringLen <= 0 || len(v) <= c.limits.maxStringLen {
		return v
	}
	c.truncated = true
	c.dropped["strings"]++
	return v[:c.limits.maxStringLen]
}

func copyDB(in []DatabaseQuery) []DatabaseQuery {
	out := make([]DatabaseQuery, len(in))
	copy(out, in)
	return out
}

func copyCache(in []CacheQuery) []CacheQuery {
	out := make([]CacheQuery, len(in))
	copy(out, in)
	return out
}

func copyLogs(in []LogEntry) []LogEntry {
	out := make([]LogEntry, len(in))
	copy(out, in)
	return out
}

func copyTimeline(in []TimelineEvent) []TimelineEvent {
	out := make([]TimelineEvent, len(in))
	copy(out, in)
	return out
}

func colorForSlow(slow bool) string {
	if slow {
		return "red"
	}
	return "blue"
}

func durationMs(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000.0
}

func unixTimestamp() float64 {
	return unixFromTime(time.Now())
}

func unixFromTime(t time.Time) float64 {
	return float64(t.UnixNano()) / 1e9
}

func toCompactString(v interface{}) string {
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func isVendorPath(path string) bool {
	if path == "" {
		return false
	}
	p := strings.ToLower(path)
	return strings.Contains(p, "/vendor/") || strings.Contains(p, "/pkg/mod/")
}

// extractTableName parses a SQL query and returns the primary table name.
func extractTableName(query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}
	tokens := strings.Fields(q)
	upper := make([]string, len(tokens))
	for i, t := range tokens {
		upper[i] = strings.ToUpper(t)
	}

	for i, tok := range upper {
		switch tok {
		case "FROM", "INTO", "UPDATE", "TABLE":
			if i+1 < len(tokens) {
				return cleanTableName(tokens[i+1])
			}
		case "JOIN":
			if i+1 < len(tokens) {
				return cleanTableName(tokens[i+1])
			}
		}
	}
	return ""
}

func cleanTableName(raw string) string {
	name := strings.Trim(raw, "`\"'[]")
	name = strings.TrimRight(name, ",;()")
	if dot := strings.LastIndex(name, "."); dot >= 0 && dot < len(name)-1 {
		name = name[dot+1:]
	}
	return name
}

// callerOutsidePackage walks the call stack starting at skip and returns the
// first frame whose file path does not contain "go-clockwork".
func callerOutsidePackage(skip int) (string, int) {
	pcs := make([]uintptr, 16)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return "", 0
	}
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.File == "" {
			if !more {
				break
			}
			continue
		}
		if !strings.Contains(frame.File, "go-clockwork") {
			return frame.File, frame.Line
		}
		if !more {
			break
		}
	}
	return "", 0
}

// contextKey is an unexported key type to prevent collisions.
type contextKey string

const collectorContextKey contextKey = "clockwork-collector"

// CollectorFromContext retrieves collector from context.
func CollectorFromContext(ctx context.Context) *Collector {
	collector, _ := ctx.Value(collectorContextKey).(*Collector)
	return collector
}

// ContextWithCollector stores collector in context.
func ContextWithCollector(ctx context.Context, collector *Collector) context.Context {
	return context.WithValue(ctx, collectorContextKey, collector)
}
