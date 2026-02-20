package sql

import (
	"context"
	"time"

	"github.com/RezaKargar/go-clockwork"
)

// Observation is a generic SQL query observation payload.
type Observation struct {
	Operation  string
	Query      string
	Duration   time.Duration
	Connection string
}

// Observer forwards SQL observations to Clockwork collectors.
type Observer struct {
	inner *clockwork.SQLObserver
}

// NewObserver creates a SQL observer.
func NewObserver(cw *clockwork.Clockwork, slowQueryThreshold time.Duration) *Observer {
	return &Observer{inner: clockwork.NewSQLObserver(cw, slowQueryThreshold)}
}

// OnQuery forwards observation into active request collector.
func (o *Observer) OnQuery(ctx context.Context, observation Observation) {
	if o == nil || o.inner == nil {
		return
	}
	o.inner.OnQuery(ctx, clockwork.QueryObservation{
		Operation:  observation.Operation,
		Query:      observation.Query,
		Duration:   observation.Duration,
		Connection: observation.Connection,
	})
}
