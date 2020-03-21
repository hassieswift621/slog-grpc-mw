package grpc_slog_test

import (
	"io"
	"runtime"
	"strings"
	"testing"
	"time"

	"cdr.dev/slog"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	pb_testproto "github.com/grpc-ecosystem/go-grpc-middleware/testing/testproto"
	grpc_slog "github.com/hassieswift621/slog-grpc-mw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func customCodeToLevel(c codes.Code) slog.Level {
	if c == codes.Unauthenticated {
		// Make this a special case for tests, and an error.
		return slog.LevelCritical
	}
	level := grpc_slog.DefaultCodeToLevel(c)
	return level
}

func TestSlogLoggingSuite(t *testing.T) {
	if strings.HasPrefix(runtime.Version(), "go1.7") {
		t.Skipf("Skipping due to json.RawMessage incompatibility with go1.7")
		return
	}
	opts := []grpc_slog.Option{
		grpc_slog.WithLevels(customCodeToLevel),
	}
	b := newBaseSlogSuite(t)
	b.InterceptorTestSuite.ServerOpts = []grpc.ServerOption{
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_slog.StreamServerInterceptor(b.log, opts...)),
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_slog.UnaryServerInterceptor(b.log, opts...)),
	}
	suite.Run(t, &slogServerSuite{b})
}

type slogServerSuite struct {
	*slogBaseSuite
}

func (s *slogServerSuite) TestPing_WithCustomTags() {
	deadline := time.Now().Add(3 * time.Second)
	_, err := s.Client.Ping(s.DeadlineCtx(deadline), goodPing)
	require.NoError(s.T(), err, "there must be not be an error on a successful call")

	msgs := s.getOutputJSONs()
	s.T().Log(msgs[0])
	s.T().Log(msgs[0]["grpc.service"])

	require.Len(s.T(), msgs, 2, "two log statements should be logged")
	for _, m := range msgs {
		// Get slog fields.
		f := m["fields"].(map[string]interface{})

		assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
		assert.Equal(s.T(), f["grpc.method"], "Ping", "all lines must contain method name")
		assert.Equal(s.T(), f["span.kind"], "server", "all lines must contain the kind of call (server)")
		assert.Equal(s.T(), f["custom_tags.string"], "something", "all lines must contain `custom_tags.string`")
		assert.Equal(s.T(), f["grpc.request.value"], "something", "all lines must contain fields extracted")
		assert.Equal(s.T(), f["custom_field"], "custom_value", "all lines must contain `custom_field`")

		assert.Contains(s.T(), f, "custom_tags.int", "all lines must contain `custom_tags.int`")
		require.Contains(s.T(), f, "grpc.start_time", "all lines must contain the start time")
		_, err := time.Parse(time.RFC3339, f["grpc.start_time"].(string))
		assert.NoError(s.T(), err, "should be able to parse start time as RFC3339")

		require.Contains(s.T(), f, "grpc.request.deadline", "all lines must contain the deadline of the call")
		_, err = time.Parse(time.RFC3339, f["grpc.request.deadline"].(string))
		require.NoError(s.T(), err, "should be able to parse deadline as RFC3339")
		assert.Equal(s.T(), f["grpc.request.deadline"], deadline.Format(time.RFC3339), "should have the same deadline that was set by the caller")
	}

	assert.Equal(s.T(), msgs[0]["msg"], "some ping", "handler's message must contain user message")

	assert.Equal(s.T(), msgs[1]["msg"], "finished unary call with code OK", "handler's message must contain user message")
	assert.Equal(s.T(), msgs[1]["level"], "INFO", "must be logged at info level")
	assert.Contains(s.T(), msgs[1]["fields"], "grpc.time_ms", "interceptor log statement should contain execution time")
}

func (s *slogServerSuite) TestPingError_WithCustomLevels() {
	for _, tcase := range []struct {
		code  codes.Code
		level slog.Level
		msg   string
	}{
		{
			code:  codes.Internal,
			level: slog.LevelError,
			msg:   "Internal must remap to LevelError in DefaultCodeToLevel",
		},
		{
			code:  codes.NotFound,
			level: slog.LevelInfo,
			msg:   "NotFound must remap to LevelInfo in DefaultCodeToLevel",
		},
		{
			code:  codes.FailedPrecondition,
			level: slog.LevelWarn,
			msg:   "FailedPrecondition must remap to LevelWarn in DefaultCodeToLevel",
		},
		{
			code:  codes.Unauthenticated,
			level: slog.LevelCritical,
			msg:   "Unauthenticated is overwritten to LevelCritical with customCodeToLevel override, which probably didn't work",
		},
	} {
		s.buffer.Reset()
		_, err := s.Client.PingError(
			s.SimpleCtx(),
			&pb_testproto.PingRequest{Value: "something", ErrorCodeReturned: uint32(tcase.code)})
		s.T().Log(err)
		require.Error(s.T(), err, "each call here must return an error")

		msgs := s.getOutputJSONs()
		require.Len(s.T(), msgs, 1, "only the interceptor log message is printed in PingErr")

		m := msgs[0]
		f := m["fields"].(map[string]interface{})
		assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
		assert.Equal(s.T(), f["grpc.method"], "PingError", "all lines must contain method name")
		assert.Equal(s.T(), f["grpc.code"], tcase.code.String(), "all lines have the correct gRPC code")
		assert.Equal(s.T(), m["level"], tcase.level.String(), tcase.msg)
		assert.Equal(s.T(), m["msg"], "finished unary call with code "+tcase.code.String(), "needs the correct end message")

		require.Contains(s.T(), f, "grpc.start_time", "all lines must contain the start time")
		_, err = time.Parse(time.RFC3339, f["grpc.start_time"].(string))
		assert.NoError(s.T(), err, "should be able to parse start time as RFC3339")
	}
}

func (s *slogServerSuite) TestPingList_WithCustomTags() {
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
	require.Len(s.T(), msgs, 2, "two log statements should be logged")

	for _, m := range msgs {
		// Get slog fields.
		f := m["fields"].(map[string]interface{})

		assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
		assert.Equal(s.T(), f["grpc.method"], "PingList", "all lines must contain method name")
		assert.Equal(s.T(), f["span.kind"], "server", "all lines must contain the kind of call (server)")
		assert.Equal(s.T(), f["custom_tags.string"], "something", "all lines must contain `custom_tags.string` set by AddFields")
		assert.Equal(s.T(), f["grpc.request.value"], "something", "all lines must contain fields extracted from goodPing because of test.manual_extractfields.pb")

		assert.Contains(s.T(), f, "custom_tags.int", "all lines must contain `custom_tags.int` set by AddFields")
		require.Contains(s.T(), f, "grpc.start_time", "all lines must contain the start time")
		_, err := time.Parse(time.RFC3339, f["grpc.start_time"].(string))
		assert.NoError(s.T(), err, "should be able to parse start time as RFC3339")
	}

	assert.Equal(s.T(), msgs[0]["msg"], "some pinglist", "handler's message must contain user message")

	assert.Equal(s.T(), msgs[1]["msg"], "finished streaming call with code OK", "handler's message must contain user message")
	assert.Equal(s.T(), msgs[1]["level"], "INFO", "OK codes must be logged on info level.")
	assert.Contains(s.T(), msgs[1]["fields"], "grpc.time_ms", "interceptor log statement should contain execution time")
}

func TestSlogLoggingOverrideSuite(t *testing.T) {
	if strings.HasPrefix(runtime.Version(), "go1.7") {
		t.Skip("Skipping due to json.RawMessage incompatibility with go1.7")
		return
	}
	opts := []grpc_slog.Option{
		grpc_slog.WithDurationField(grpc_slog.DurationToDurationField),
	}
	b := newBaseSlogSuite(t)
	b.InterceptorTestSuite.ServerOpts = []grpc.ServerOption{
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_slog.StreamServerInterceptor(b.log, opts...)),
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_slog.UnaryServerInterceptor(b.log, opts...)),
	}
	suite.Run(t, &slogServerOverrideSuite{b})
}

type slogServerOverrideSuite struct {
	*slogBaseSuite
}

func (s *slogServerOverrideSuite) TestPing_HasOverriddenDuration() {
	_, err := s.Client.Ping(s.SimpleCtx(), goodPing)
	require.NoError(s.T(), err, "there must be not be an error on a successful call")
	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 2, "two log statements should be logged")

	for _, m := range msgs {
		// Get slog fields.
		f := m["fields"].(map[string]interface{})

		assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
		assert.Equal(s.T(), f["grpc.method"], "Ping", "all lines must contain method name")
	}

	// Get slog fields.
	f0 := msgs[0]["fields"].(map[string]interface{})
	f1 := msgs[1]["fields"].(map[string]interface{})

	assert.Equal(s.T(), msgs[0]["msg"], "some ping", "handler's message must contain user message")
	assert.NotContains(s.T(), f0, "grpc.time_ms", "handler's message must not contain default duration")
	assert.NotContains(s.T(), f0, "grpc.duration", "handler's message must not contain overridden duration")

	assert.Equal(s.T(), msgs[1]["msg"], "finished unary call with code OK", "handler's message must contain user message")
	assert.Equal(s.T(), msgs[1]["level"], "INFO", "OK error codes must be logged on info level.")
	assert.NotContains(s.T(), f1, "grpc.time_ms", "handler's message must not contain default duration")
	assert.Contains(s.T(), f1, "grpc.duration", "handler's message must contain overridden duration")
}

func (s *slogServerOverrideSuite) TestPingList_HasOverriddenDuration() {
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
	require.Len(s.T(), msgs, 2, "two log statements should be logged")
	for _, m := range msgs {
		s.T()
		// Get slog fields.
		f := m["fields"].(map[string]interface{})
		assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
		assert.Equal(s.T(), f["grpc.method"], "PingList", "all lines must contain method name")
	}

	// Get slog fields.
	f0 := msgs[0]["fields"].(map[string]interface{})
	f1 := msgs[1]["fields"].(map[string]interface{})

	assert.Equal(s.T(), msgs[0]["msg"], "some pinglist", "handler's message must contain user message")
	assert.NotContains(s.T(), f0, "grpc.time_ms", "handler's message must not contain default duration")
	assert.NotContains(s.T(), f0, "grpc.duration", "handler's message must not contain overridden duration")

	assert.Equal(s.T(), msgs[1]["msg"], "finished streaming call with code OK", "handler's message must contain user message")
	assert.Equal(s.T(), msgs[1]["level"], "INFO", "OK error codes must be logged on info level.")
	assert.NotContains(s.T(), f1, "grpc.time_ms", "handler's message must not contain default duration")
	assert.Contains(s.T(), f1, "grpc.duration", "handler's message must contain overridden duration")
}

func TestSlogServerOverrideSuppressedSuite(t *testing.T) {
	if strings.HasPrefix(runtime.Version(), "go1.7") {
		t.Skip("Skipping due to json.RawMessage incompatibility with go1.7")
		return
	}
	opts := []grpc_slog.Option{
		grpc_slog.WithDecider(func(method string, err error) bool {
			if err != nil && method == "/mwitkow.testproto.TestService/PingError" {
				return true
			}
			return false
		}),
	}
	b := newBaseSlogSuite(t)
	b.InterceptorTestSuite.ServerOpts = []grpc.ServerOption{
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_slog.StreamServerInterceptor(b.log, opts...)),
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_slog.UnaryServerInterceptor(b.log, opts...)),
	}
	suite.Run(t, &slogServerOverriddenDeciderSuite{b})
}

type slogServerOverriddenDeciderSuite struct {
	*slogBaseSuite
}

func (s *slogServerOverriddenDeciderSuite) TestPing_HasOverriddenDecider() {
	_, err := s.Client.Ping(s.SimpleCtx(), goodPing)
	require.NoError(s.T(), err, "there must be not be an error on a successful call")
	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 1, "single log statements should be logged")

	// Get slog fields.
	f := msgs[0]["fields"].(map[string]interface{})

	assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
	assert.Equal(s.T(), f["grpc.method"], "Ping", "all lines must contain method name")
	assert.Equal(s.T(), msgs[0]["msg"], "some ping", "handler's message must contain user message")
}

func (s *slogServerOverriddenDeciderSuite) TestPingError_HasOverriddenDecider() {
	code := codes.NotFound
	level := slog.LevelInfo
	msg := "NotFound must remap to LevelInfo in DefaultCodeToLevel"

	s.buffer.Reset()
	_, err := s.Client.PingError(
		s.SimpleCtx(),
		&pb_testproto.PingRequest{Value: "something", ErrorCodeReturned: uint32(code)})
	require.Error(s.T(), err, "each call here must return an error")
	msgs := s.getOutputJSONs()
	require.Len(s.T(), msgs, 1, "only the interceptor log message is printed in PingErr")
	m := msgs[0]
	// Get slog fields.
	f := m["fields"].(map[string]interface{})
	assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
	assert.Equal(s.T(), f["grpc.method"], "PingError", "all lines must contain method name")
	assert.Equal(s.T(), f["grpc.code"], code.String(), "all lines must contain the correct gRPC code")
	assert.Equal(s.T(), m["level"], level.String(), msg)
}

func (s *slogServerOverriddenDeciderSuite) TestPingList_HasOverriddenDecider() {
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
	require.Len(s.T(), msgs, 1, "single log statements should be logged")

	// Get slog fields.
	f := msgs[0]["fields"].(map[string]interface{})

	assert.Equal(s.T(), f["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
	assert.Equal(s.T(), f["grpc.method"], "PingList", "all lines must contain method name")
	assert.Equal(s.T(), msgs[0]["msg"], "some pinglist", "handler's message must contain user message")

	assert.NotContains(s.T(), f, "grpc.time_ms", "handler's message must not contain default duration")
	assert.NotContains(s.T(), f, "grpc.duration", "handler's message must not contain overridden duration")
}
