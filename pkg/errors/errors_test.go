package errors

import (
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoTranscriptFoundError_Error(t *testing.T) {
	err := &NoTranscriptFoundError{
		Languages: []string{"fr", "de"},
		Available: []string{"en", "es"},
	}
	msg := err.Error()
	assert.Contains(t, msg, "fr")
	assert.Contains(t, msg, "de")
	assert.Contains(t, msg, "en")
	assert.Contains(t, msg, "es")
}

func TestTranscriptsDisabledError_Error(t *testing.T) {
	err := &TranscriptsDisabledError{VideoID: "abc123"}
	assert.Contains(t, err.Error(), "abc123")
}

func TestNoTranscriptFoundError_ErrorsAs(t *testing.T) {
	original := &NoTranscriptFoundError{
		Languages: []string{"fr"},
		Available: []string{"en", "es"},
	}
	wrapped := fmt.Errorf("outer: %w", original)

	var target *NoTranscriptFoundError
	assert.True(t, stderrors.As(wrapped, &target))
	assert.Equal(t, []string{"fr"}, target.Languages)
	assert.Equal(t, []string{"en", "es"}, target.Available)
}

func TestTranscriptsDisabledError_ErrorsAs(t *testing.T) {
	original := &TranscriptsDisabledError{VideoID: "abc123"}
	wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", original))

	var target *TranscriptsDisabledError
	assert.True(t, stderrors.As(wrapped, &target))
	assert.Equal(t, "abc123", target.VideoID)
}
