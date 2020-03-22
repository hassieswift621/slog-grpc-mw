package grpc_slog

import (
	"context"
	"path"
	"time"

	"google.golang.org/grpc"

	"cdr.dev/slog"
)

var (
	// ClientField is used in every client-side log statement made through grpc_slog. Can be overwritten before initialization.
	ClientField = slog.Field{"span.kind", "client"}
)

// UnaryClientInterceptor returns a new unary client interceptor that optionally logs the execution of external gRPC calls.
func UnaryClientInterceptor(logger slog.Logger, opts ...Option) grpc.UnaryClientInterceptor {
	o := evaluateClientOpt(opts)
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		fields := newClientLoggerFields(ctx, method)
		startTime := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		logFinalClientLine(ctx, o, logger.With(fields...), startTime, err, "finished client unary call")
		return err
	}
}

// StreamClientInterceptor returns a new streaming client interceptor that optionally logs the execution of external gRPC calls.
func StreamClientInterceptor(logger slog.Logger, opts ...Option) grpc.StreamClientInterceptor {
	o := evaluateClientOpt(opts)
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		fields := newClientLoggerFields(ctx, method)
		startTime := time.Now()
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		logFinalClientLine(ctx, o, logger.With(fields...), startTime, err, "finished client streaming call")
		return clientStream, err
	}
}

func logFinalClientLine(ctx context.Context, o *options, logger slog.Logger, startTime time.Time, err error, msg string) {
	code := o.codeFunc(err)
	level := o.levelFunc(code)
	log(ctx, logger, level, msg,
		slog.Error(err),
		slog.F("grpc.code", code.String()),
		o.durationFunc(time.Now().Sub(startTime)),
	)
}

func newClientLoggerFields(ctx context.Context, fullMethodString string) []slog.Field {
	service := path.Dir(fullMethodString)[1:]
	method := path.Base(fullMethodString)
	return []slog.Field{
		SystemField,
		ClientField,
		slog.F("grpc.service", service),
		slog.F("grpc.method", method),
	}
}
