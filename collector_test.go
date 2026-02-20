package clockwork

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCollector_RespectsLimitsAndMarksTruncated(t *testing.T) {
	collector := NewCollector("GET", "/products", collectorLimits{
		maxRequestBytes: 10 * 1024,
		maxStringLen:    8,
		maxLogs:         1,
	})

	require.NotNil(t, collector)

	collector.AddLogEntry("info", "this-message-is-too-long", map[string]interface{}{"key": "value"})
	collector.AddLogEntry("info", "second", nil)

	meta := collector.GetMetadata()
	require.NotNil(t, meta)
	require.True(t, meta.Truncated)
	require.NotEmpty(t, meta.Dropped)
	require.GreaterOrEqual(t, meta.Dropped["logs"], 1)
	require.GreaterOrEqual(t, meta.Dropped["strings"], 1)
}

func TestCollector_ComputesDatabaseDuration(t *testing.T) {
	collector := NewCollector("GET", "/db", collectorLimits{})
	require.NotNil(t, collector)

	collector.AddDatabaseQuery("SELECT 1", 20*time.Millisecond, "mysql", false)
	collector.AddDatabaseQuery("SELECT 2", 30*time.Millisecond, "mysql", true)

	meta := collector.GetMetadata()
	require.NotNil(t, meta)
	require.Len(t, meta.DatabaseQueries, 2)
	require.InDelta(t, 50.0, meta.DatabaseDuration, 5.0)
}
