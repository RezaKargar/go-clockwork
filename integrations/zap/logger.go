package zap

import (
	"go.uber.org/zap"
)

// Logger adapts *zap.Logger to clockwork.Logger, so it can be passed directly
// to middleware constructors (e.g. clockworkgin.Setup) for their own internal
// warnings (failed metadata saves, missing metadata lookups). *zap.Logger
// itself doesn't satisfy clockwork.Logger - its Warn takes ...zap.Field, not
// clockwork.Logger's untyped ...interface{} - so some adapter is required;
// this is that adapter, kept in the zap integration rather than clockwork's
// dependency-free core.
type Logger struct {
	inner *zap.Logger
}

// NewLogger wraps l so it satisfies clockwork.Logger.
func NewLogger(l *zap.Logger) *Logger {
	return &Logger{inner: l}
}

func (z *Logger) Warn(msg string, keysAndValues ...interface{}) {
	if z.inner == nil {
		return
	}
	fields := make([]zap.Field, 0, len(keysAndValues)/2)
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key, _ := keysAndValues[i].(string)
		fields = append(fields, zap.Any(key, keysAndValues[i+1]))
	}
	z.inner.Warn(msg, fields...)
}
