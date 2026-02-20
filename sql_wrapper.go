package clockwork

import (
	"context"
	"time"
)

// QueryObservation is a generic SQL query observation.
type QueryObservation struct {
	Operation  string
	Query      string
	Duration   time.Duration
	Connection string
}

// SQLObserver bridges query observations into active Clockwork collectors.
type SQLObserver struct {
	cw                 *Clockwork
	slowQueryThreshold time.Duration
}

// NewSQLObserver creates a SQL observer for Clockwork.
func NewSQLObserver(cw *Clockwork, slowQueryThreshold time.Duration) *SQLObserver {
	if cw == nil || !cw.IsEnabled() {
		return nil
	}
	if slowQueryThreshold <= 0 {
		slowQueryThreshold = cw.config.SlowQueryThreshold
	}
	return &SQLObserver{cw: cw, slowQueryThreshold: slowQueryThreshold}
}

// OnQuery receives SQL observations.
func (o *SQLObserver) OnQuery(ctx context.Context, observation QueryObservation) {
	if o == nil || o.cw == nil {
		return
	}

	collector := CollectorFromContext(ctx)
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
	collector.AddDatabaseQuery(query, observation.Duration, conn, slow)
}
