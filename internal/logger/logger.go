// Zerolog middleware
package logger

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type loggerKey struct{}

// GetLogger returns the attached logger from the context, or the global logger if not set
func GetLogger(ctx context.Context) *zerolog.Logger {
	if logger := ctx.Value(loggerKey{}); logger != nil {
		return logger.(*zerolog.Logger)
	}
	return &log.Logger
}

// GetGlobalLogger returns the global logger, useful if a context is not available
func GetGlobalLogger() *zerolog.Logger {
	return &log.Logger
}

// WithLogger adds a logger to a context
func WithLogger(parent context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(parent, loggerKey{}, &logger)
}

// Wrapper to map go-retryablehttp logger
type RetryableLogger struct{}

func (*RetryableLogger) Error(msg string, keysAndValues ...interface{}) {
	logRetryable(GetGlobalLogger().Warn(), msg, keysAndValues) // Downgrade
}
func (*RetryableLogger) Info(msg string, keysAndValues ...interface{}) {
	logRetryable(GetGlobalLogger().Debug(), msg, keysAndValues) // Downgrade
}
func (*RetryableLogger) Debug(msg string, keysAndValues ...interface{}) {
	logRetryable(GetGlobalLogger().Trace(), msg, keysAndValues) // Downgrade
}
func (*RetryableLogger) Warn(msg string, keysAndValues ...interface{}) {
	logRetryable(GetGlobalLogger().Warn(), msg, keysAndValues) // Downgrade
}

func logRetryable(e *zerolog.Event, msg string, keysAndValues []interface{}) {
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		if i+1 >= len(keysAndValues) {
			continue
		}
		e.Interface(key, keysAndValues[i+1])
	}
	e.Bool("is_retryable_http_log", true)
	e.Msg(msg)
}
