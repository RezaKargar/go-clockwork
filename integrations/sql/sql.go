package sql

import (
	"context"
	"time"

	"github.com/RezaKargar/go-clockwork"
)

// Observation is a SQL query observation payload.
type Observation struct {
	Operation  string
	Query      string
	Duration   time.Duration
	Connection string
	Model      string // table or model name; auto-extracted from SQL if empty
	File       string // caller file path
	Line       int    // caller line number
}

// Observer forwards SQL observations to Clockwork collectors.
type Observer struct {
	cw                 *clockwork.Clockwork
	slowQueryThreshold time.Duration
}

// NewObserver creates a SQL observer that records queries to the active request collector.
func NewObserver(cw *clockwork.Clockwork, slowQueryThreshold time.Duration) *Observer {
	if cw == nil || !cw.IsEnabled() {
		return nil
	}
	if slowQueryThreshold <= 0 {
		slowQueryThreshold = cw.Config().SlowQueryThreshold
	}
	return &Observer{cw: cw, slowQueryThreshold: slowQueryThreshold}
}

// OnQuery records the observation on the active request collector from context.
func (o *Observer) OnQuery(ctx context.Context, observation Observation) {
	if o == nil || o.cw == nil {
		return
	}

	collector := clockwork.CollectorFromContext(ctx)
	if collector == nil {
		return
	}

	query := observation.Query
	if query == "" {
		query = observation.Operation
	}
	conn := observation.Connection
	if conn == "" {
		conn = "sql"
	}

	slow := observation.Duration > o.slowQueryThreshold
	collector.AddDatabaseQueryDetailed(
		query, observation.Duration, conn, slow,
		observation.Model, observation.File, observation.Line,
	)
}
