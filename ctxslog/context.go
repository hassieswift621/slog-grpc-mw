package ctxslog

import (
	"context"
	"io/ioutil"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
)

type ctxMarker struct{}

type ctxLogger struct {
	logger slog.Logger
	fields []slog.Field
}

var (
	ctxMarkerKey = &ctxMarker{}
)

// AddFields adds fields to the logger.
func AddFields(ctx context.Context, fields ...slog.Field) {
	l, ok := ctx.Value(ctxMarkerKey).(*ctxLogger)
	if !ok || l == nil {
		return
	}
	l.fields = append(l.fields, fields...)
}

// Extract takes the call-scoped Logger from grpc_slog middleware.
//
// It always returns a Logger that has all the grpc_ctxtags updated.
func Extract(ctx context.Context) slog.Logger {
	l, ok := ctx.Value(ctxMarkerKey).(*ctxLogger)
	if !ok || l == nil {
		return sloghuman.Make(ioutil.Discard)
	}
	// Add grpc_ctxtags tags metadata until now.
	fields := TagsToFields(ctx)
	return l.logger.With(fields...)
}

// TagsToFields transforms the Tags on the supplied context into slog fields.
func TagsToFields(ctx context.Context) []slog.Field {
	var fields []slog.Field
	tags := grpc_ctxtags.Extract(ctx)
	for k, v := range tags.Values() {
		fields = append(fields, slog.Field{Name: k, Value: v})
	}
	return fields
}

// ToContext adds the slog.Logger to the context for extraction later.
// Returning the new context that has been created.
func ToContext(ctx context.Context, logger slog.Logger) context.Context {
	l := &ctxLogger{
		logger: logger,
	}
	return context.WithValue(ctx, ctxMarkerKey, l)
}
