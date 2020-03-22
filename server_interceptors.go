package grpc_slog

import (
	"context"
	"path"
	"time"

	"github.com/hassieswift621/slog-grpc-mw/ctxslog"

	"cdr.dev/slog"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
)

var (
	// SystemField is used in every log statement made through grpc_slog. Can be overwritten before any initialization code.
	SystemField = slog.Field{Name: "system", Value: "grpc"}
	// ServerField is used in every server-side log statement made through grpc_slog. Can be overwritten before initialization.
	ServerField = slog.Field{Name: "span.kind", Value: "server"}
)

// UnaryServerInterceptor returns a new unary server interceptors that adds slog.Logger to the context.
func UnaryServerInterceptor(logger slog.Logger, opts ...Option) grpc.UnaryServerInterceptor {
	o := evaluateServerOpt(opts)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		newCtx := newLoggerForCall(ctx, logger, info.FullMethod, startTime)

		resp, err := handler(newCtx, req)
		if !o.shouldLog(info.FullMethod, err) {
			return resp, err
		}
		code := o.codeFunc(err)
		level := o.levelFunc(code)

		// re-extract logger from newCtx, as it may have extra fields that changed in the holder.
		extractedLogger := ctxslog.Extract(newCtx)
		log(ctx, extractedLogger, level, "finished unary call with code "+code.String(),
			slog.Error(err),
			slog.F("grpc.code", code.String()),
			o.durationFunc(time.Since(startTime)),
		)

		return resp, err
	}
}

// StreamServerInterceptor returns a new streaming server interceptor that adds slog.Logger to the context.
func StreamServerInterceptor(logger slog.Logger, opts ...Option) grpc.StreamServerInterceptor {
	o := evaluateServerOpt(opts)
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()
		newCtx := newLoggerForCall(stream.Context(), logger, info.FullMethod, startTime)
		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = newCtx

		err := handler(srv, wrapped)
		if !o.shouldLog(info.FullMethod, err) {
			return err
		}
		code := o.codeFunc(err)
		level := o.levelFunc(code)

		// re-extract logger from newCtx, as it may have extra fields that changed in the holder.
		extractedLogger := ctxslog.Extract(newCtx)
		log(stream.Context(), extractedLogger, level, "finished streaming call with code "+code.String(),
			slog.Error(err),
			slog.F("grpc.code", code.String()),
			o.durationFunc(time.Since(startTime)),
		)

		return err
	}
}

func serverCallFields(fullMethodString string) []slog.Field {
	service := path.Dir(fullMethodString)[1:]
	method := path.Base(fullMethodString)
	return []slog.Field{
		SystemField,
		ServerField,
		{"grpc.service", service},
		{"grpc.method", method},
	}
}

func newLoggerForCall(ctx context.Context, logger slog.Logger, fullMethodString string, start time.Time) context.Context {
	var f []slog.Field
	f = append(f, slog.F("grpc.start_time", start.Format(time.RFC3339)))
	if d, ok := ctx.Deadline(); ok {
		f = append(f, slog.F("grpc.request.deadline", d.Format(time.RFC3339)))
	}
	callLog := logger.With(append(f, serverCallFields(fullMethodString)...)...)
	return ctxslog.ToContext(ctx, callLog)
}
