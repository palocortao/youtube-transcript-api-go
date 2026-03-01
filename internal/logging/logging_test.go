package logging

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromContext_ReturnsDefaultWhenNotSet(t *testing.T) {
	assert.Equal(t, slog.Default(), FromContext(context.Background()))
}

func TestWithLogger_RoundTrip(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := WithLogger(context.Background(), logger)
	assert.Equal(t, logger, FromContext(ctx))
}

func TestFromContext_NilLogger_FallsBackToDefault(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextKey{}, (*slog.Logger)(nil))
	assert.Equal(t, slog.Default(), FromContext(ctx))
}

func TestWithLogger_DoesNotAffectParentContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parent := context.Background()
	_ = WithLogger(parent, logger)
	assert.Equal(t, slog.Default(), FromContext(parent))
}
