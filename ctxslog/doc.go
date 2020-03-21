/*
`ctxslog` is a ctxlogger that is backed by slog

It accepts a user-configured `slog.Logger` that will be used for logging. The same `slog.Logger` will
be populated into the `context.Context` passed into gRPC handler code.

You can use `ctxslog.Extract` to log into a request-scoped `slog.Logger` instance in your handler code.

As `ctxslog.Extract` will iterate all tags on from `grpc_ctxtags` it is therefore expensive so it is advised that you
extract once at the start of the function from the context and reuse it for the remainder of the function (see examples).

Please see examples and tests for examples of use.
*/
package ctxslog
