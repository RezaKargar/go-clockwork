package clockwork

import (
	"context"
)

// DataSource is a pluggable source that can contribute data to a request's metadata
// when the request completes. Register with Clockwork.RegisterDataSource.
// Implement this interface to add custom data (e.g. memory stats, custom metrics)
// that will be available in Metadata.UserData or via the collector's existing methods.
type DataSource interface {
	Name() string
	Resolve(ctx context.Context, collector DataCollector)
}
