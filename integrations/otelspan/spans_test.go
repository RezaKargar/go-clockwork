package otelspan

import (
	"context"
	"testing"

	clockworkpkg "github.com/RezaKargar/go-clockwork"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func newTestCollector(cfg Config) (*clockworkpkg.Clockwork, *clockworkpkg.Collector, *sdktrace.TracerProvider) {
	cw := clockworkpkg.NewClockwork(clockworkpkg.DefaultConfig(), nil)
	sp := NewSpanProcessor(cw, cfg)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sp))
	collector := cw.NewCollector("GET", "/products")
	return cw, collector, tp
}

func TestSpanProcessorForwardsEventsAndAttributesToLogs(t *testing.T) {
	_, collector, tp := newTestCollector(Config{})
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")
	ctx := clockworkpkg.ContextWithCollector(context.Background(), collector)

	_, span := tracer.Start(ctx, "internal.work")
	span.SetAttributes(attribute.String("widget.id", "brands"))
	span.AddEvent("search.backend.force_es")
	span.End()

	_, dbSpan := tracer.Start(ctx, "db.query")
	dbSpan.SetAttributes(
		attribute.String("db.system", "mysql"),
		attribute.String("db.statement", "SELECT 1"),
		attribute.String("db.custom_field", "value"),
	)
	dbSpan.End()

	meta := collector.GetMetadata()

	foundEvent, foundInternalAttr, foundDBAttr := false, false, false
	for _, entry := range meta.LogEntries {
		if entry.Message == "internal.work: search.backend.force_es" {
			foundEvent = true
		}
		if entry.Message == "internal.work" && entry.Context["widget.id"] == "brands" {
			foundInternalAttr = true
		}
		if entry.Message == "db.query" && entry.Context["db.custom_field"] == "value" {
			foundDBAttr = true
		}
	}

	if !foundEvent {
		t.Errorf("expected a log entry for the span event, got entries: %+v", meta.LogEntries)
	}
	if !foundInternalAttr {
		t.Errorf("expected a log entry with the internal span's attributes, got entries: %+v", meta.LogEntries)
	}
	if !foundDBAttr {
		t.Errorf("expected a log entry with the db span's custom attribute, got entries: %+v", meta.LogEntries)
	}
	if len(meta.DatabaseQueries) != 1 || meta.DatabaseQueries[0].Query != "SELECT 1" {
		t.Errorf("expected db.query to still be classified into DatabaseQueries, got: %+v", meta.DatabaseQueries)
	}
}

func TestSpanProcessorClassifiesCacheSpansWithDefaultConfig(t *testing.T) {
	_, collector, tp := newTestCollector(Config{})
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")
	ctx := clockworkpkg.ContextWithCollector(context.Background(), collector)

	_, hit := tracer.Start(ctx, "cache.Get")
	hit.SetAttributes(attribute.Bool("cache.hit", true))
	hit.End()

	_, miss := tracer.Start(ctx, "cache.Get")
	miss.SetAttributes(attribute.Bool("cache.hit", false))
	miss.End()

	meta := collector.GetMetadata()
	if len(meta.CacheQueries) != 2 {
		t.Fatalf("expected 2 cache queries, got %+v", meta.CacheQueries)
	}
	if meta.CacheQueries[0].Type != "hit" || meta.CacheQueries[1].Type != "miss" {
		t.Errorf("expected hit then miss, got %+v", meta.CacheQueries)
	}
}

func TestSpanProcessorHonorsCustomCacheConfig(t *testing.T) {
	_, collector, tp := newTestCollector(Config{
		CacheSpanPrefix: "mycache.",
		CacheHitAttr:    "mycache.hit",
	})
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")
	ctx := clockworkpkg.ContextWithCollector(context.Background(), collector)

	_, span := tracer.Start(ctx, "mycache.Get")
	span.SetAttributes(attribute.Bool("mycache.hit", true))
	span.End()

	meta := collector.GetMetadata()
	if len(meta.CacheQueries) != 1 || meta.CacheQueries[0].Type != "hit" {
		t.Fatalf("expected 1 hit cache query with custom config, got %+v", meta.CacheQueries)
	}
}

func TestSpanProcessorClassifiesHTTPSpans(t *testing.T) {
	_, collector, tp := newTestCollector(Config{})
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")
	ctx := clockworkpkg.ContextWithCollector(context.Background(), collector)

	_, span := tracer.Start(ctx, "GET")
	span.SetAttributes(
		attribute.String("http.request.method", "GET"),
		attribute.String("url.full", "https://example.com/x"),
		attribute.Int64("http.response.status_code", 204),
	)
	span.End()

	meta := collector.GetMetadata()
	if len(meta.HTTPRequests) != 1 {
		t.Fatalf("expected 1 http request, got %+v", meta.HTTPRequests)
	}
	if meta.HTTPRequests[0].Response == nil || meta.HTTPRequests[0].Response.Status != 204 {
		t.Errorf("expected status 204, got %+v", meta.HTTPRequests[0].Response)
	}
}
