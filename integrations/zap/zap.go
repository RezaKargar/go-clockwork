package zap

import (
	"github.com/RezaKargar/go-clockwork"
	"go.uber.org/zap/zapcore"
)

// WrapCore wraps a zap core and mirrors correlated logs into Clockwork collectors.
func WrapCore(core zapcore.Core, cw *clockwork.Clockwork) zapcore.Core {
	return clockwork.WrapZapCore(core, cw)
}
