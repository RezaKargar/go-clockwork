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

func TestExtractTableName(t *testing.T) {
	tests := []struct {
		sql   string
		table string
	}{
		{"SELECT * FROM users WHERE id = ?", "users"},
		{"select id, name from `products` where active = 1", "products"},
		{"INSERT INTO orders (user_id, total) VALUES (?, ?)", "orders"},
		{"UPDATE categories SET name = ? WHERE id = ?", "categories"},
		{"DELETE FROM sessions WHERE expired_at < ?", "sessions"},
		{"SELECT u.id FROM schema.users u JOIN orders o ON u.id = o.user_id", "users"},
		{"", ""},
		{"SHOW TABLES", ""},
	}
	for _, tt := range tests {
		t.Run(tt.sql, func(t *testing.T) {
			require.Equal(t, tt.table, extractTableName(tt.sql))
		})
	}
}

func TestCollector_AddDatabaseQueryExtractsModel(t *testing.T) {
	collector := NewCollector("GET", "/test", collectorLimits{})
	collector.AddDatabaseQuery("SELECT * FROM products WHERE id = 1", 5*time.Millisecond, "mysql", false)

	meta := collector.GetMetadata()
	require.Len(t, meta.DatabaseQueries, 1)
	require.Equal(t, "products", meta.DatabaseQueries[0].Model)
}

func TestCollector_AddDatabaseQueryDetailedUsesExplicitModel(t *testing.T) {
	collector := NewCollector("GET", "/test", collectorLimits{})
	collector.AddDatabaseQueryDetailed(
		"SELECT * FROM products WHERE id = 1", 5*time.Millisecond, "mysql", false,
		"ProductModel", "app/handlers/product.go", 42,
	)

	meta := collector.GetMetadata()
	require.Len(t, meta.DatabaseQueries, 1)
	require.Equal(t, "ProductModel", meta.DatabaseQueries[0].Model)
	require.Equal(t, "app/handlers/product.go", meta.DatabaseQueries[0].File)
	require.Equal(t, 42, meta.DatabaseQueries[0].Line)
}
