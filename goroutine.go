package clockwork

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// stackBufPool amortises the []byte allocation used by CurrentGoroutineID.
var stackBufPool = sync.Pool{
	New: func() any { return make([]byte, 64) },
}

// CurrentGoroutineID extracts the calling goroutine's ID from runtime.Stack
// output. It exists so context-free call sites (like zapcore.Core, which
// receives no context.Context) can still be correlated with the Collector
// handling the request on that goroutine.
func CurrentGoroutineID() uint64 {
	buf := stackBufPool.Get().([]byte)
	n := runtime.Stack(buf, false)
	if n >= len(buf) {
		buf = make([]byte, 128)
		n = runtime.Stack(buf, false)
	}
	s := string(buf[:n])
	stackBufPool.Put(buf)

	const prefix = "goroutine "
	if !strings.HasPrefix(s, prefix) {
		return 0
	}
	s = s[len(prefix):]
	if i := strings.IndexByte(s, ' '); i > 0 {
		s = s[:i]
	}
	id, _ := strconv.ParseUint(s, 10, 64)
	return id
}
