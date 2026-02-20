package clockwork

import (
	"math"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
)

// ClockworkCore wraps a zap core and mirrors correlated logs into Clockwork collectors.
type ClockworkCore struct {
	underlying zapcore.Core
	cw         *Clockwork
	fields     []zapcore.Field
}

// WrapZapCore wraps a zap core for Clockwork log collection.
func WrapZapCore(core zapcore.Core, cw *Clockwork) zapcore.Core {
	if core == nil || cw == nil || !cw.IsEnabled() {
		return core
	}
	return &ClockworkCore{
		underlying: core,
		cw:         cw,
	}
}

func (c *ClockworkCore) Enabled(level zapcore.Level) bool {
	return c.underlying.Enabled(level)
}

func (c *ClockworkCore) With(fields []zapcore.Field) zapcore.Core {
	next := &ClockworkCore{
		underlying: c.underlying.With(fields),
		cw:         c.cw,
		fields:     append(append(make([]zapcore.Field, 0, len(c.fields)+len(fields)), c.fields...), fields...),
	}
	return next
}

func (c *ClockworkCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(entry.Level) {
		return checked
	}
	return checked.AddCore(entry, c)
}

func (c *ClockworkCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	err := c.underlying.Write(entry, fields)

	if c.cw == nil || !c.cw.HasActiveTraces() {
		return err
	}

	traceID := findTraceID(c.fields)
	if traceID == "" {
		traceID = findTraceID(fields)
	}

	contextFields := make(map[string]interface{}, 8)
	appendFields(contextFields, c.fields)
	appendFields(contextFields, fields)
	if entry.LoggerName != "" {
		contextFields["logger"] = entry.LoggerName
	}
	if !entry.Time.IsZero() {
		contextFields["time"] = entry.Time.UTC().Format(time.RFC3339Nano)
	}

	traceFrames := buildLogTrace(entry)

	if traceID == "" {
		c.cw.RecordLogForSingleActiveWithTrace(entry.Level.String(), entry.Message, contextFields, traceFrames)
		return err
	}

	c.cw.RecordLogForTraceWithTrace(traceID, entry.Level.String(), entry.Message, contextFields, traceFrames)
	return err
}

func (c *ClockworkCore) Sync() error {
	return c.underlying.Sync()
}

func findTraceID(fields []zapcore.Field) string {
	for i := range fields {
		if fields[i].Key == "trace_id" && fields[i].Type == zapcore.StringType {
			return fields[i].String
		}
	}
	return ""
}

func appendFields(dst map[string]interface{}, fields []zapcore.Field) {
	if len(dst) >= 20 {
		return
	}
	for i := range fields {
		if len(dst) >= 20 {
			return
		}
		field := fields[i]
		if field.Key == "trace_id" {
			continue
		}
		if value, ok := fieldValue(field); ok {
			dst[field.Key] = value
		}
	}
}

func fieldValue(field zapcore.Field) (interface{}, bool) {
	switch field.Type {
	case zapcore.StringType:
		return field.String, true
	case zapcore.BoolType:
		return field.Integer == 1, true
	case zapcore.Int64Type:
		return field.Integer, true
	case zapcore.Int32Type:
		return int32(field.Integer), true
	case zapcore.Int16Type:
		return int16(field.Integer), true
	case zapcore.Int8Type:
		return int8(field.Integer), true
	case zapcore.Uint64Type:
		return uint64(field.Integer), true
	case zapcore.Uint32Type:
		return uint32(field.Integer), true
	case zapcore.Uint16Type:
		return uint16(field.Integer), true
	case zapcore.Uint8Type:
		return uint8(field.Integer), true
	case zapcore.Float64Type:
		return math.Float64frombits(uint64(field.Integer)), true
	case zapcore.Float32Type:
		return float32(math.Float64frombits(uint64(field.Integer))), true
	case zapcore.DurationType:
		return time.Duration(field.Integer).String(), true
	case zapcore.TimeType:
		if field.Interface != nil {
			if loc, ok := field.Interface.(*time.Location); ok {
				return time.Unix(0, field.Integer).In(loc).Format(time.RFC3339Nano), true
			}
		}
		return time.Unix(0, field.Integer).UTC().Format(time.RFC3339Nano), true
	case zapcore.ErrorType:
		if field.Interface == nil {
			return "", true
		}
		if err, ok := field.Interface.(error); ok {
			return err.Error(), true
		}
		return toCompactString(field.Interface), true
	case zapcore.ByteStringType:
		if field.Interface == nil {
			return "", true
		}
		if b, ok := field.Interface.([]byte); ok {
			return string(b), true
		}
		return "", false
	default:
		if field.Interface != nil {
			return toCompactString(field.Interface), true
		}
		return "", false
	}
}

func buildLogTrace(entry zapcore.Entry) []LogTraceFrame {
	frames := make([]LogTraceFrame, 0, maxLogTraceFrames)

	if entry.Caller.Defined && entry.Caller.File != "" {
		frames = append(frames, LogTraceFrame{
			File:     entry.Caller.File,
			Line:     entry.Caller.Line,
			IsVendor: isVendorPath(entry.Caller.File),
		})
	}

	parsed := parseZapStack(entry.Stack)
	if len(parsed) == 0 {
		return frames
	}

	for _, frame := range parsed {
		if len(frames) >= maxLogTraceFrames {
			break
		}
		if isSameFrame(frame, frames) {
			continue
		}
		frames = append(frames, frame)
	}

	return frames
}

func parseZapStack(stack string) []LogTraceFrame {
	stack = strings.TrimSpace(stack)
	if stack == "" {
		return nil
	}

	lines := strings.Split(stack, "\n")
	frames := make([]LogTraceFrame, 0, maxLogTraceFrames)
	for i := 0; i < len(lines)-1 && len(frames) < maxLogTraceFrames; i++ {
		call := strings.TrimSpace(lines[i])
		if call == "" {
			continue
		}

		fileLine := strings.TrimSpace(lines[i+1])
		file, line, ok := parseFileLine(fileLine)
		if !ok {
			continue
		}
		frames = append(frames, LogTraceFrame{
			Call:     call,
			File:     file,
			Line:     line,
			IsVendor: isVendorPath(file),
		})
		i++
	}
	return frames
}

func parseFileLine(in string) (string, int, bool) {
	if in == "" {
		return "", 0, false
	}

	in = strings.TrimSuffix(in, ")")
	idx := strings.LastIndex(in, ":")
	if idx == -1 || idx == len(in)-1 {
		return "", 0, false
	}

	file := in[:idx]
	lineStr := in[idx+1:]
	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return "", 0, false
	}
	return file, line, true
}

func isSameFrame(candidate LogTraceFrame, existing []LogTraceFrame) bool {
	for _, frame := range existing {
		if frame.File == candidate.File && frame.Line == candidate.Line && frame.Call == candidate.Call {
			return true
		}
	}
	return false
}
