module github.com/RezaKargar/go-clockwork/integrations/sql

go 1.26

require github.com/RezaKargar/go-clockwork v0.2.0

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
)

replace github.com/RezaKargar/go-clockwork => ../..
