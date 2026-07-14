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

	// HTTPRequests holds outbound HTTP calls this request made to other
	// services (i.e. routes called externally), shown in the viewer's
	// "HTTP Requests" tab.
	HTTPRequests []HTTPRequest `json:"httpRequests"`

	LogEntries []LogEntry `json:"log"`

	TimelineEvents []TimelineEvent `json:"timelineData"`

	MemoryUsage uint64         `json:"memoryUsage"`
	Truncated   bool           `json:"truncated,omitempty"`
	Dropped     map[string]int `json:"dropped,omitempty"`

	// UserData holds arbitrary data from DataSource implementations and custom integrations.
	UserData map[string]interface{} `json:"userData,omitempty"`
}

// DatabaseQuery represents a database query in Clockwork payload.
//
// Time is when the query started, as Unix seconds (matching Metadata.Time's
// epoch-seconds convention) - the Clockwork viewer positions this event on
// the timeline as [Time, Time+Duration/1000]. Duration is in milliseconds.
type DatabaseQuery struct {
	Query      string  `json:"query"`
	Duration   float64 `json:"duration"`
	Connection string  `json:"connection"`
	Model      string  `json:"model,omitempty"`
	File       string  `json:"file,omitempty"`
	Line       int     `json:"line,omitempty"`
	Slow       bool    `json:"slow"`
	Time       float64 `json:"time"`
}

// CacheQuery represents a cache operation in Clockwork payload.
//
// Time is when the operation started, as Unix seconds; see DatabaseQuery.Time.
type CacheQuery struct {
	Type     string  `json:"type"`
	Key      string  `json:"key"`
	Duration float64 `json:"duration"`
	Time     float64 `json:"time"`
}

// HTTPRequest represents an outbound HTTP call in Clockwork payload.
//
// Time is when the call started, as Unix seconds; see DatabaseQuery.Time.
// Request.URL is the full raw URL string - the viewer parses it into
// scheme/host/path/query client-side, so it must not be pre-split here.
type HTTPRequest struct {
	Time     float64           `json:"time"`
	Duration float64           `json:"duration"`
	Request  HTTPRequestInfo   `json:"request"`
	Response *HTTPResponseInfo `json:"response,omitempty"`
}

// HTTPRequestInfo describes the outbound request in an HTTPRequest entry.
type HTTPRequestInfo struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

// HTTPResponseInfo describes the response in an HTTPRequest entry, when known.
type HTTPResponseInfo struct {
	Status int `json:"status"`
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
