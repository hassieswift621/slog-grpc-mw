package grpc_slog

import (
	"context"
	"path"

	"cdr.dev/slog"
)

var (
	// ClientField is used in every client-side log statement made through grpc_slog. Can be overwritten before initialization.
	ClientField = slog.Field{"span.kind", "client"}
)

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
