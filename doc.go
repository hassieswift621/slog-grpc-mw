/*
`grpc_slog` is a gRPC logging middleware backed by Slog loggers

It accepts a user-configured `slog.Logger` that will be used for logging completed gRPC calls. The same `slog.Logger` will
be used for logging completed gRPC calls, and be populated into the `context.Context` passed into gRPC handler code.

On calling `StreamServerInterceptor` or `UnaryServerInterceptor` this logging middleware will add gRPC call information
to the ctx so that it will be present on subsequent use of the `ctx_slog` logger.

If a deadline is present on the gRPC request the grpc.request.deadline tag is populated when the request begins. grpc.request.deadline
is a string representing the time (RFC3339) when the current call will expire.

This package also implements request and response *payload* logging, both for server-side and client-side. These will be
logged as structured `jsonpb` fields for every message received/sent (both unary and streaming). For that please use
`Payload*Interceptor` functions. Please note that the user-provided function that determines whether to log
the full request/response payload needs to be written with care, as this can significantly slow down gRPC.

Slog can also be made as a backend for gRPC library internals. For that use `ReplaceGrpcLoggerV2`.

*Server Interceptor*
Below is a JSON formatted example of a log that would be logged by the server interceptor:
	{
		"level": "INFO",							// string	Slog log levels
		"msg": "finished unary call with code OK",	// string	log message
		"ts": "2020-04-08T19:16:33.7517136Z",		// string	Slog RFC3339 log timestamp
		"caller": "slog-grpc-mw/shared_test.go:30",	// string	Slog caller - file
		"func":	"github.com/hassieswift621/slog-grpc-mw_test.(*loggingPingService).Ping",	// string	Slog caller - func
		"fields": {								// object	Slog fields
			"custom_field": "custom_value",		// string	user defined field
			"custom_tags.int": 1337,			// int	user defined tag on the ctx
			"custom_tags.string": "something",	// string	user defined tag on the ctx
			"grpc.code": "OK",					// string	grpc status code
			"grpc.method": "Ping",				// string	method name
			"grpc.service": "mwitkow.testproto.TestService", 		// string	full name of the called service
			"grpc.request.deadline": "2006-01-02T15:04:05Z07:00",	// string	RFC3339 deadline of the current request if supplied
			"grpc.request.value": "something",						// string	value on the request
			"grpc.service": "mwitkow.testproto.TestService",		// string	full name of the called service
			"grpc.start_time": "2006-01-02T15:04:05Z07:00",			// string	RFC3339 representation of the start time
			"grpc.time_ms": 1.234,									// float32	run time of the call in ms
			"peer.address": "127.0.0.1:55948"						// string	IP address of calling party and the port which the call is incoming on
			"span.kind": "server",									// string	client | server
			"system": "grpc"										// string
		}
	}

*Payload Interceptor*
Below is a JSON formatted example of a log that would be logged by the payload interceptor:
	{
		"level": "INFO",	// string	Slog log levels
		"msg": "client request payload logged as grpc.request.content",			// string	log message
		"caller": "slog-grpc-mw/payload_interceptors.go:130",					// string	Slog caller - file
		"func": "github.com/hassieswift621/slog-grpc-mw.logProtoMessageAsJson",	// string	Slog caller - func
		"fields": {						// object	Slog fields
			"grpc.request.content": {	// object	content of RPC request
				"sleepTimeMs": 9999,	// int		defined by caller
				"value": something,		// string	defined by caller
			},
			"grpc.method": "Ping",								// string	method name
			"grpc.service": "mwitkow.testproto.TestService",	// string	full name of the called service
			"span.kind": "server",								// string	client | server
			"system": "grpc"									// string
		}
	}

Please see examples and tests for examples of use.
*/
package grpc_slog
