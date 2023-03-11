package fh

import (
	"fmt"

	"go.uber.org/zap"
)

// InternalLogger a logger for fasthttp.Logger
type InternalLogger struct {
	logger *zap.Logger
}

func NewInternalLogger(logger *zap.Logger) *InternalLogger {
	logger = logger.WithOptions(
		zap.WithCaller(false),
		zap.AddStacktrace(zap.PanicLevel))

	return &InternalLogger{
		logger: logger,
	}
}

func (i *InternalLogger) Printf(format string, args ...interface{}) {
	i.logger.Error("fasthttp log", zap.String("msg", fmt.Sprintf(format, args...)))
}
