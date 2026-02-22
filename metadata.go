package clockwork

// Metadata represents the stored Clockwork payload for a request.
// Field names follow Clockwork's expected JSON shape where practical.
type Metadata struct {
	ID               string  `json:"id"`
	Version          int     `json:"version"`
	Type             string  `json:"type,omitempty"`
	Time             float64 `json:"time"`
	ResponseTime     float64 `json:"responseTime"`
	ResponseStatus   int     `json:"responseStatus"`
	ResponseDuration float64 `json:"responseDuration"`

	Method     string            `json:"method"`
	URI        string            `json:"uri"`
	URL        string            `json:"url,omitempty"`
	Controller string            `json:"controller,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`

	TraceID string `json:"traceId,omitempty"`
	SpanID  string `json:"spanId,omitempty"`

	DatabaseQueries      []DatabaseQuery `json:"databaseQueries"`
	DatabaseQueriesCount int             `json:"databaseQueriesCount"`
	DatabaseDuration     float64         `json:"databaseDuration"`

	CacheQueries []CacheQuery `json:"cacheQueries"`
	LogEntries   []LogEntry   `json:"log"`

	TimelineEvents []TimelineEvent `json:"timelineData"`

	MemoryUsage uint64         `json:"memoryUsage"`
	Truncated   bool           `json:"truncated,omitempty"`
	Dropped     map[string]int `json:"dropped,omitempty"`

	// UserData holds arbitrary data from DataSource implementations and custom integrations.
	UserData map[string]interface{} `json:"userData,omitempty"`
}

// DatabaseQuery represents a database query in Clockwork payload.
type DatabaseQuery struct {
	Query      string  `json:"query"`
	Duration   float64 `json:"duration"`
	Connection string  `json:"connection"`
	Slow       bool    `json:"slow"`
	Timestamp  float64 `json:"timestamp"`
}

// CacheQuery represents a cache operation in Clockwork payload.
type CacheQuery struct {
	Type      string  `json:"type"`
	Key       string  `json:"key"`
	Duration  float64 `json:"duration"`
	Timestamp float64 `json:"timestamp"`
}

// LogEntry represents a log message in Clockwork payload.
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp float64                `json:"time"`
	Trace     []LogTraceFrame        `json:"trace,omitempty"`
}

// LogTraceFrame represents one stack frame for a log entry.
type LogTraceFrame struct {
	Call     string `json:"call,omitempty"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	IsVendor bool   `json:"isVendor"`
}

// TimelineEvent represents a timeline event in Clockwork payload.
type TimelineEvent struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Start       float64 `json:"start"`
	End         float64 `json:"end,omitempty"`
	Duration    float64 `json:"duration,omitempty"`
	Color       string  `json:"color,omitempty"`
}
