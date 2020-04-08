package grpc_slog

import (
	"cdr.dev/slog"
	"context"
	"fmt"
	"google.golang.org/grpc/grpclog"
)

type slogGrpcLoggerV2 struct {
	logger    slog.Logger
	verbosity int
}

// ReplaceGrpcLoggerV2 replaces the grpc_log.LoggerV2 with the provided logger.
func ReplaceGrpcLoggerV2(logger slog.Logger) {
	ReplaceGrpcLoggerV2WithVerbosity(logger, 0)
}

// ReplaceGrpcLoggerV2WithVerbosity replaces the grpc_.LoggerV2 with the provided logger and verbosity.
func ReplaceGrpcLoggerV2WithVerbosity(logger slog.Logger, verbosity int) {
	grpcLogger := &slogGrpcLoggerV2{
		logger:    logger.With(SystemField, slog.F("grpc_log", true)),
		verbosity: 0,
	}
	grpclog.SetLoggerV2(grpcLogger)
}

func (l *slogGrpcLoggerV2) Info(args ...interface{}) {
	l.logger.Info(context.Background(), fmt.Sprint(args...))
}

func (l *slogGrpcLoggerV2) Infoln(args ...interface{}) {
	l.logger.Info(context.Background(), fmt.Sprint(args...))
}

func (l *slogGrpcLoggerV2) Infof(format string, args ...interface{}) {
	l.logger.Info(context.Background(), fmt.Sprintf(format, args...))
}

func (l *slogGrpcLoggerV2) Warning(args ...interface{}) {
	l.logger.Warn(context.Background(), fmt.Sprint(args...))
}

func (l *slogGrpcLoggerV2) Warningln(args ...interface{}) {
	l.logger.Warn(context.Background(), fmt.Sprint(args...))
}

func (l *slogGrpcLoggerV2) Warningf(format string, args ...interface{}) {
	l.logger.Warn(context.Background(), fmt.Sprintf(format, args...))
}

func (l *slogGrpcLoggerV2) Error(args ...interface{}) {
	l.logger.Error(context.Background(), fmt.Sprint(args...))
}

func (l *slogGrpcLoggerV2) Errorln(args ...interface{}) {
	l.logger.Error(context.Background(), fmt.Sprint(args...))
}

func (l *slogGrpcLoggerV2) Errorf(format string, args ...interface{}) {
	l.logger.Error(context.Background(), fmt.Sprintf(format, args...))
}

func (l *slogGrpcLoggerV2) Fatal(args ...interface{}) {
	l.logger.Fatal(context.Background(), fmt.Sprint(args...))
}

func (l *slogGrpcLoggerV2) Fatalln(args ...interface{}) {
	l.logger.Fatal(context.Background(), fmt.Sprint(args...))
}

func (l *slogGrpcLoggerV2) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal(context.Background(), fmt.Sprintf(format, args...))
}

func (l *slogGrpcLoggerV2) V(level int) bool {
	return level <= l.verbosity
}
