package grpc_slog_test

import (
	"context"
	"io/ioutil"
	"time"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	pb_testproto "github.com/grpc-ecosystem/go-grpc-middleware/testing/testproto"
	grpc_slog "github.com/hassieswift621/slog-grpc-mw"
	"github.com/hassieswift621/slog-grpc-mw/ctxslog"
	"google.golang.org/grpc"
)

var (
	logger     slog.Logger
	customFunc grpc_slog.CodeToLevel
)

// Initialization shows a relatively complex initialization sequence.
func Example_initialization() {
	// Shared options for the logger, with a custom gRPC code to log level function.
	opts := []grpc_slog.Option{
		grpc_slog.WithLevels(customFunc),
	}

	// Make sure that log statements internal to gRPC library are logged using the slog logger as well.
	grpc_slog.ReplaceGrpcLoggerV2(logger)

	// Create a server, make sure we put the grpc_ctxtags context before everything else.
	_ = grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_slog.UnaryServerInterceptor(logger, opts...),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_slog.StreamServerInterceptor(logger, opts...),
		),
	)
}

// Initialization shows an initialization sequence with the duration field generation overridden.
func Example_initializationWithDurationFieldOverride() {
	opts := []grpc_slog.Option{
		grpc_slog.WithDurationField(func(duration time.Duration) slog.Field {
			return slog.F("grpc.time_ns", duration.Nanoseconds())
		}),
	}

	_ = grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_slog.UnaryServerInterceptor(logger, opts...),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_slog.StreamServerInterceptor(logger, opts...),
		),
	)
}

// Simple unary handler that adds custom fields to the requests's context. These will be used for all log statements.
func ExampleExtract_unary() {
	_ = func(ctx context.Context, ping *pb_testproto.PingRequest) (*pb_testproto.PingResponse, error) {
		// Add fields the ctxtags of the request which will be added to all extracted loggers.
		grpc_ctxtags.Extract(ctx).Set("custom_tags.string", "something").Set("custom_tags.int", 1337)

		// Extract a single request-scoped slog.Logger and log messages. (containing the grpc.xxx tags)
		l := ctxslog.Extract(ctx)
		l.Info(ctx, "some ping")
		l.Info(ctx, "another ping")
		return &pb_testproto.PingResponse{Value: ping.Value}, nil
	}
}

func Example_initializationWithDecider() {
	opts := []grpc_slog.Option{
		grpc_slog.WithDecider(func(fullMethodName string, err error) bool {
			// will not log gRPC calls if it was a call to healthcheck and no error was raised
			if err == nil && fullMethodName == "foo.bar.healthcheck" {
				return false
			}

			// by default everything will be logged
			return true
		}),
	}

	// Initialise nop logger.
	nopLogger := sloghuman.Make(ioutil.Discard)

	_ = []grpc.ServerOption{
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_slog.StreamServerInterceptor(nopLogger, opts...)),
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_slog.UnaryServerInterceptor(nopLogger, opts...)),
	}
}
