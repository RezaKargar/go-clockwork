// Package otelspan bridges an application's existing OpenTelemetry spans into
// Clockwork, as an alternative to wrapping pkg/sql, pkg/cache, etc. directly:
// if the app already produces OTel spans (via otelsql, an OTel-instrumented
// cache wrapper, otelhttp, ...), SpanProcessor reads them and classifies them
// into Clockwork's Database/Cache/HTTP Requests/Timeline/Logs tabs, so no
// second, parallel instrumentation layer is needed.
package otelspan

import (
	"context"
	"strconv"
	"strings"
	"sync"

	clockworkpkg "github.com/RezaKargar/go-clockwork"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Config controls how SpanProcessor classifies spans into Clockwork's tabs.
// A zero-value Config is valid: NewSpanProcessor fills in defaults for any
// unset field via withDefaults.
type Config struct {
	// CacheSpanPrefix is the span-name prefix that marks a span as a cache
	// operation, routed to the Cache tab (e.g. a span named
	// CacheSpanPrefix+"Get" is a cache read). Defaults to "cache.". Set to a
	// value that can never match a real span name (e.g. "\x00") to disable
	// cache-span classification - such spans then fall through to Timeline.
	CacheSpanPrefix string

	// CacheHitAttr is the boolean attribute key on a cache Get-shaped span
	// (CacheSpanPrefix+"Get") indicating a hit (true) or miss (false).
	// Defaults to "cache.hit".
	CacheHitAttr string

	// MaxTimelineAttrs caps how many span attributes are rendered into a
	// Timeline event's description, keeping the line readable in the
	// Clockwork UI. Defaults to 10.
	MaxTimelineAttrs int
}

// DefaultConfig returns the Config NewSpanProcessor uses for any zero-value field.
func DefaultConfig() Config {
	return Config{
		CacheSpanPrefix:  "cache.",
		CacheHitAttr:     "cache.hit",
		MaxTimelineAttrs: 10,
	}
}

func (c Config) withDefaults() Config {
	d := DefaultConfig()
	if c.CacheSpanPrefix == "" {
		c.CacheSpanPrefix = d.CacheSpanPrefix
	}
	if c.CacheHitAttr == "" {
		c.CacheHitAttr = d.CacheHitAttr
	}
	if c.MaxTimelineAttrs <= 0 {
		c.MaxTimelineAttrs = d.MaxTimelineAttrs
	}
	return c
}

// SpanProcessor is an OTel SpanProcessor that feeds completed spans into the
// Clockwork collector. It classifies spans by their attributes:
//   - name has the configured cache prefix → cache operation (Cache tab)
//   - http.request.method attr            → outbound HTTP call (HTTP Requests tab)
//   - db.system present                   → database query (Database tab)
//   - everything else                      → timeline event (Timeline tab)
//
// Every span, regardless of classification, also has its OTel events and
// full attribute set forwarded to the Logs tab (see logSpanEvents) - the
// classification above only ever extracts the small, specific attribute
// subset each destination needs, so this is the only place nothing is lost.
//
// Server spans are skipped for classification, since a request-level
// middleware typically already records the request's own timeline event;
// their events/attributes are still forwarded to Logs.
//
// It also registers the collector against whichever goroutine starts each
// span (see cw.RegisterGoroutine), not just the top-level request goroutine
// a middleware registers, so log lines from worker goroutines fanned out for
// parallel work are captured too (see clockworkpkg.CurrentGoroutineID).
type SpanProcessor struct {
	mu            sync.Mutex
	collectors    map[trace.SpanID]*clockworkpkg.Collector
	spanGoroutine map[trace.SpanID]uint64
	cw            *clockworkpkg.Clockwork
	slowThreshold int64 // milliseconds; 0 means no slow-query detection
	cfg           Config
}

// NewSpanProcessor creates a SpanProcessor that feeds cw. Any zero-value
// field in cfg falls back to DefaultConfig(); pass Config{} to use every
// default.
func NewSpanProcessor(cw *clockworkpkg.Clockwork, cfg Config) *SpanProcessor {
	var threshold int64
	if cw != nil {
		threshold = cw.Config().SlowQueryThreshold.Milliseconds()
	}
	return &SpanProcessor{
		collectors:    make(map[trace.SpanID]*clockworkpkg.Collector),
		spanGoroutine: make(map[trace.SpanID]uint64),
		cw:            cw,
		slowThreshold: threshold,
		cfg:           cfg.withDefaults(),
	}
}

func (p *SpanProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	collector := clockworkpkg.CollectorFromContext(parent)
	if collector == nil {
		return
	}
	spanID := s.SpanContext().SpanID()
	goroutineID := clockworkpkg.CurrentGoroutineID()

	p.mu.Lock()
	p.collectors[spanID] = collector
	p.spanGoroutine[spanID] = goroutineID
	p.mu.Unlock()

	p.cw.RegisterGoroutine(goroutineID, collector)
}

func (p *SpanProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	spanID := s.SpanContext().SpanID()

	p.mu.Lock()
	collector, ok := p.collectors[spanID]
	delete(p.collectors, spanID)
	goroutineID, hasGoroutine := p.spanGoroutine[spanID]
	delete(p.spanGoroutine, spanID)
	p.mu.Unlock()

	if hasGoroutine {
		p.cw.UnregisterGoroutine(goroutineID)
	}

	if !ok || collector == nil {
		return
	}

	logSpanEvents(collector, s)

	if s.SpanKind() == trace.SpanKindServer {
		return
	}

	dur := s.EndTime().Sub(s.StartTime())
	attrs := s.Attributes()

	// Cache spans also carry a db.system attribute (semconv.DBSystemCache /
	// DBSystemMemcached) to stay semantic-convention compliant, so this check
	// must run before the generic db.system check below or every cache
	// operation is misclassified as a database query.
	if strings.HasPrefix(s.Name(), p.cfg.CacheSpanPrefix) {
		key := findAttr(attrs, "db.statement")
		cacheType := p.classifyCacheOp(s.Name(), attrs)
		collector.AddCacheQuery(cacheType, key, dur)
		return
	}

	if dbSystem := findAttr(attrs, "db.system"); dbSystem != "" {
		query := findAttr(attrs, "db.statement")
		if query == "" {
			query = s.Name()
		}
		isSlow := p.slowThreshold > 0 && dur.Milliseconds() > p.slowThreshold
		collector.AddDatabaseQuery(query, dur, dbSystem, isSlow)
		return
	}

	// Outbound HTTP calls to other services (otelhttp client spans carry
	// http.request.method; semconv v1.40.0+) - these are "routes called
	// externally", shown in the viewer's HTTP Requests tab.
	if method := findAttr(attrs, "http.request.method"); method != "" {
		url := findAttr(attrs, "url.full")
		status := 0
		if code := findAttr(attrs, "http.response.status_code"); code != "" {
			status, _ = strconv.Atoi(code)
		}
		collector.AddHTTPRequest(method, url, status, dur)
		return
	}

	color := spanKindColor(s.SpanKind())
	if s.Status().Code == codes.Error {
		color = "red"
	}
	collector.AddTimelineEvent(
		s.Name(),
		describeSpan(s, p.cfg.MaxTimelineAttrs),
		s.StartTime(),
		s.EndTime(),
		color,
	)
}

// logSpanEvents forwards a span's OTel events and full attribute set into
// Clockwork's Logs tab. It runs for every span, including the server span
// skipped above, because span events (explicit `span.AddEvent(...)`
// breadcrumbs) and a span's own attributes carry information the
// classification above doesn't preserve - those destinations only ever
// forward the small, specific attribute subset each display needs.
func logSpanEvents(collector *clockworkpkg.Collector, s sdktrace.ReadOnlySpan) {
	for _, ev := range s.Events() {
		level := "info"
		if ev.Name == "exception" {
			level = "error"
		}
		collector.AddLogEntry(level, s.Name()+": "+ev.Name, attrsToMap(ev.Attributes))
	}

	if fields := attrsToMap(s.Attributes()); fields != nil {
		collector.AddLogEntry("debug", s.Name(), fields)
	}
}

func attrsToMap(attrs []attribute.KeyValue) map[string]interface{} {
	if len(attrs) == 0 {
		return nil
	}
	m := make(map[string]interface{}, len(attrs))
	for _, a := range attrs {
		m[string(a.Key)] = attrValue(a.Value)
	}
	return m
}

func attrValue(v attribute.Value) interface{} {
	switch v.Type() {
	case attribute.BOOL:
		return v.AsBool()
	case attribute.INT64:
		return v.AsInt64()
	case attribute.FLOAT64:
		return v.AsFloat64()
	case attribute.STRING:
		return v.AsString()
	default:
		return v.AsString()
	}
}

// describeSpan renders a span's kind, attributes, and error status into a
// single-line description, since Clockwork's TimelineEvent has no dedicated
// field for arbitrary span attributes. maxAttrs caps how many attributes are
// included.
func describeSpan(s sdktrace.ReadOnlySpan, maxAttrs int) string {
	parts := make([]string, 0, maxAttrs+2)
	parts = append(parts, s.SpanKind().String())

	if status := s.Status(); status.Code == codes.Error {
		if status.Description != "" {
			parts = append(parts, "error="+status.Description)
		} else {
			parts = append(parts, "error=true")
		}
	}

	attrs := s.Attributes()
	for i, a := range attrs {
		if i >= maxAttrs {
			break
		}
		parts = append(parts, string(a.Key)+"="+attrValueString(a.Value))
	}

	return strings.Join(parts, " ")
}

func attrValueString(v attribute.Value) string {
	switch v.Type() {
	case attribute.BOOL:
		return strconv.FormatBool(v.AsBool())
	case attribute.INT64:
		return strconv.FormatInt(v.AsInt64(), 10)
	case attribute.FLOAT64:
		return strconv.FormatFloat(v.AsFloat64(), 'g', -1, 64)
	case attribute.STRING:
		return v.AsString()
	default:
		return v.AsString()
	}
}

func (p *SpanProcessor) Shutdown(context.Context) error   { return nil }
func (p *SpanProcessor) ForceFlush(context.Context) error { return nil }

func findAttr(attrs []attribute.KeyValue, key string) string {
	for _, a := range attrs {
		if string(a.Key) == key {
			return attrValueString(a.Value)
		}
	}
	return ""
}

func findBoolAttr(attrs []attribute.KeyValue, key string) (bool, bool) {
	for _, a := range attrs {
		if string(a.Key) == key {
			return a.Value.AsBool(), true
		}
	}
	return false, false
}

// classifyCacheOp maps a cache span's operation suffix (the span name with
// cfg.CacheSpanPrefix stripped) to a Cache-tab operation type. Get/Set/
// SetMany/Delete are a fixed vocabulary - only the prefix and hit-attribute
// key are configurable via Config.
func (p *SpanProcessor) classifyCacheOp(spanName string, attrs []attribute.KeyValue) string {
	op := strings.TrimPrefix(spanName, p.cfg.CacheSpanPrefix)
	switch op {
	case "Get":
		if hit, ok := findBoolAttr(attrs, p.cfg.CacheHitAttr); ok && hit {
			return "hit"
		}
		return "miss"
	case "Set", "SetMany":
		return "write"
	case "Delete":
		return "delete"
	default:
		return "read"
	}
}

func spanKindColor(kind trace.SpanKind) string {
	switch kind {
	case trace.SpanKindClient:
		return "blue"
	case trace.SpanKindInternal:
		return "gray"
	case trace.SpanKindProducer:
		return "purple"
	case trace.SpanKindConsumer:
		return "orange"
	default:
		return ""
	}
}
