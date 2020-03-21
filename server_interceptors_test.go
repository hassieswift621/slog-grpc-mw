package grpc_slog_test

import (
	"cdr.dev/slog"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_slog "github.com/hassieswift621/slog-grpc-mw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"runtime"
	"strings"
	"testing"
	"time"
)

func customCodeToLevel(c codes.Code) slog.Level {
	if c == codes.Unauthenticated {
		// Make this a special case for tests, and an error.
		return slog.LevelFatal
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
	for _, msg := range msgs {
		// Get slog fields.
		m := msg["fields"].(map[string]interface{})

		assert.Equal(s.T(), m["grpc.service"], "mwitkow.testproto.TestService", "all lines must contain service name")
		assert.Equal(s.T(), m["grpc.method"], "Ping", "all lines must contain method name")
		assert.Equal(s.T(), m["span.kind"], "server", "all lines must contain the kind of call (server)")
		assert.Equal(s.T(), m["custom_tags.string"], "something", "all lines must contain `custom_tags.string`")
		assert.Equal(s.T(), m["grpc.request.value"], "something", "all lines must contain fields extracted")
		assert.Equal(s.T(), m["custom_field"], "custom_value", "all lines must contain `custom_field`")

		assert.Contains(s.T(), m, "custom_tags.int", "all lines must contain `custom_tags.int`")
		require.Contains(s.T(), m, "grpc.start_time", "all lines must contain the start time")
		_, err := time.Parse(time.RFC3339, m["grpc.start_time"].(string))
		assert.NoError(s.T(), err, "should be able to parse start time as RFC3339")

		require.Contains(s.T(), m, "grpc.request.deadline", "all lines must contain the deadline of the call")
		_, err = time.Parse(time.RFC3339, m["grpc.request.deadline"].(string))
		require.NoError(s.T(), err, "should be able to parse deadline as RFC3339")
		assert.Equal(s.T(), m["grpc.request.deadline"], deadline.Format(time.RFC3339), "should have the same deadline that was set by the caller")
	}

	assert.Equal(s.T(), msgs[0]["msg"], "some ping", "handler's message must contain user message")

	assert.Equal(s.T(), msgs[1]["msg"], "finished unary call with code OK", "handler's message must contain user message")
	assert.Equal(s.T(), msgs[1]["level"], "INFO", "must be logged at info level")
	assert.Contains(s.T(), msgs[1]["fields"], "grpc.time_ms", "interceptor log statement should contain execution time")
}
