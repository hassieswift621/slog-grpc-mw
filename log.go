package grpc_slog

import (
	"context"

	"cdr.dev/slog"
)

// log logs based on the log level.
func log(ctx context.Context, logger slog.Logger, level slog.Level, msg string, fields ...slog.Field) {
	switch level {
	case slog.LevelDebug:
		logger.Debug(ctx, msg, fields...)
	case slog.LevelInfo:
		logger.Info(ctx, msg, fields...)
	case slog.LevelWarn:
		logger.Warn(ctx, msg, fields...)
	case slog.LevelError:
		logger.Error(ctx, msg, fields...)
	case slog.LevelCritical:
		logger.Critical(ctx, msg, fields...)
	case slog.LevelFatal:
		logger.Fatal(ctx, msg, fields...)
	}
}
