package grpc_slog_test

import (
	"io"
	"runtime"
	"strings"
	"testing"

	"cdr.dev/slog"
	pb_testproto "github.com/grpc-ecosystem/go-grpc-middleware/testing/testproto"
	grpc_slog "github.com/hassieswift621/slog-grpc-mw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func customClientCodeToLevel(c codes.Code) slog.Level {
	if c == codes.Unauthenticated {
		// Make this a special case for tests, and an error.
		return slog.LevelError
	}
	level := grpc_slog.DefaultClientCodeToLevel(c)
	return level
}

func TestSlogClientSuite(t *testing.T) {
	if strings.HasPrefix(runtime.Version(), "go1.7") {
		t.Skipf("Skipping due to json.RawMessage incompatibility with go1.7")
		return
	}
	opts := []grpc_slog.Option{
		grpc_slog.WithLevels(customClientCodeToLevel),
	}
	b := newBaseSlogSuite(t)
	b.log = b.log.Leveled(slog.LevelDebug)
	b.InterceptorTestSuite.ClientOpts = []grpc.DialOption{
		grpc.WithUnaryInterceptor(grpc_slog.UnaryClientInterceptor(b.log, opts...)),
		grpc.WithStreamInterceptor(grpc_slog.StreamClientInterceptor(b.log, opts...)),
	}
	suite.Run(t, &slogClientSuite{b})
}

type slogClientSuite struct {
	*slogBaseSuite
}

func (s *slogClientSuite) TestPing() {
	_, err := s.Client.Ping(s.SimpleCtx(), goodPing)
	require.NoError(s.T(), err, "there must be not be an error on a successful call")

	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 1, "one log statement should be logged")

	// Get slog fields.
	f := msgs[0]["fields"].(map[string]interface{})

	assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
	assert.Equal(s.T(), f["grpc.method"], "Ping", "all lines must contain method name")
	assert.Equal(s.T(), msgs[0]["msg"], "finished client unary call", "must contain correct message")
	assert.Equal(s.T(), f["span.kind"], "client", "all lines must contain the kind of call (client)")
	assert.Equal(s.T(), msgs[0]["level"], "DEBUG", "must be logged on debug level.")
	assert.Contains(s.T(), f, "grpc.time_ms", "interceptor log statement should contain execution time")
}

func (s *slogClientSuite) TestPingList() {
	stream, err := s.Client.PingList(s.SimpleCtx(), goodPing)
	require.NoError(s.T(), err, "should not fail on establishing the stream")
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(s.T(), err, "reading stream should not fail")
	}
	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 1, "one log statement should be logged")

	// Get slog fields.
	f := msgs[0]["fields"].(map[string]interface{})

	assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
	assert.Equal(s.T(), f["grpc.method"], "PingList", "all lines must contain method name")
	assert.Equal(s.T(), msgs[0]["msg"], "finished client streaming call", "handler's message must contain user message")
	assert.Equal(s.T(), f["span.kind"], "client", "all lines must contain the kind of call (client)")
	assert.Equal(s.T(), msgs[0]["level"], "DEBUG", "OK codes must be logged on debug level.")
	assert.Contains(s.T(), f, "grpc.time_ms", "handler's message must contain time in ms")
}

func (s *slogClientSuite) TestPingError_WithCustomLevels() {
	for _, tcase := range []struct {
		code  codes.Code
		level slog.Level
		msg   string
	}{
		{
			code:  codes.Internal,
			level: slog.LevelWarn,
			msg:   "Internal must remap to LevelWarn in DefaultClientCodeToLevel",
		},
		{
			code:  codes.NotFound,
			level: slog.LevelDebug,
			msg:   "NotFound must remap to LevelDebug in DefaultClientCodeToLevel",
		},
		{
			code:  codes.FailedPrecondition,
			level: slog.LevelDebug,
			msg:   "FailedPrecondition must remap to LevelDebug in DefaultClientCodeToLevel",
		},
		{
			code:  codes.Unauthenticated,
			level: slog.LevelError,
			msg:   "Unauthenticated is overwritten to LevelError with customClientCodeToLevel override, which probably didn't work",
		},
	} {
		s.SetupTest()
		_, err := s.Client.PingError(
			s.SimpleCtx(),
			&pb_testproto.PingRequest{Value: "something", ErrorCodeReturned: uint32(tcase.code)})
		require.Error(s.T(), err, "each call here must return an error")

		msgs := s.getOutputJSONs()
		require.Len(s.T(), msgs, 1, "only the interceptor log message is printed in PingErr")

		// Get slog fields.
		f := msgs[0]["fields"].(map[string]interface{})

		assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
		assert.Equal(s.T(), f["grpc.method"], "PingError", "all lines must contain method name")
		assert.Equal(s.T(), f["grpc.code"], tcase.code.String(), "all lines must contain the correct gRPC code")
		assert.Equal(s.T(), msgs[0]["level"], tcase.level.String(), tcase.msg)
	}
}

func TestSlogClientOverrideSuite(t *testing.T) {
	if strings.HasPrefix(runtime.Version(), "go1.7") {
		t.Skip("Skipping due to json.RawMessage incompatibility with go1.7")
		return
	}
	opts := []grpc_slog.Option{
		grpc_slog.WithDurationField(grpc_slog.DurationToDurationField),
	}
	b := newBaseSlogSuite(t)
	b.log = b.log.Leveled(slog.LevelDebug)
	b.InterceptorTestSuite.ClientOpts = []grpc.DialOption{
		grpc.WithUnaryInterceptor(grpc_slog.UnaryClientInterceptor(b.log, opts...)),
		grpc.WithStreamInterceptor(grpc_slog.StreamClientInterceptor(b.log, opts...)),
	}
	suite.Run(t, &slogClientOverrideSuite{b})
}

type slogClientOverrideSuite struct {
	*slogBaseSuite
}

func (s *slogClientOverrideSuite) TestPing_HasOverrides() {
	_, err := s.Client.Ping(s.SimpleCtx(), goodPing)
	require.NoError(s.T(), err, "there must be not be an error on a successful call")

	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 1, "one log statement should be logged")

	// Get slog fields.
	f := msgs[0]["fields"].(map[string]interface{})

	assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
	assert.Equal(s.T(), f["grpc.method"], "Ping", "all lines must contain method name")
	assert.Equal(s.T(), msgs[0]["msg"], "finished client unary call", "handler's message must contain user message")

	assert.NotContains(s.T(), f, "grpc.time_ms", "handler's message must not contain default duration")
	assert.Contains(s.T(), f, "grpc.duration", "handler's message must contain overridden duration")
}

func (s *slogClientOverrideSuite) TestPingList_HasOverrides() {
	stream, err := s.Client.PingList(s.SimpleCtx(), goodPing)
	require.NoError(s.T(), err, "should not fail on establishing the stream")
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(s.T(), err, "reading stream should not fail")
	}
	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 1, "one log statement should be logged")

	// Get slog fields.
	f := msgs[0]["fields"].(map[string]interface{})

	assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
	assert.Equal(s.T(), f["grpc.method"], "PingList", "all lines must contain method name")
	assert.Equal(s.T(), msgs[0]["msg"], "finished client streaming call", "handler's message must contain user message")
	assert.Equal(s.T(), f["span.kind"], "client", "all lines must contain the kind of call (client)")
	assert.Equal(s.T(), msgs[0]["level"], "DEBUG", "must be logged on debug level.")

	assert.NotContains(s.T(), f, "grpc.time_ms", "handler's message must not contain default duration")
	assert.Contains(s.T(), f, "grpc.duration", "handler's message must contain overridden duration")
}
