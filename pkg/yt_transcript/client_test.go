package yt_transcript

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/palocortao/youtube-transcript-api-go/internal/logging"
	"github.com/palocortao/youtube-transcript-api-go/pkg/yt_transcript_formatters"
	"github.com/palocortao/youtube-transcript-api-go/pkg/yt_transcript_models"
	"github.com/stretchr/testify/assert"
)

// mockService captures calls made to it for assertion.
type mockService struct {
	transcripts []yt_transcript_models.Transcript
	err         error
	capturedCtx context.Context
}

func (m *mockService) GetTranscripts(ctx context.Context, videoID string, languages []string, preserveFormatting bool) ([]yt_transcript_models.Transcript, error) {
	m.capturedCtx = ctx
	return m.transcripts, m.err
}

var sampleTranscripts = []yt_transcript_models.Transcript{
	{
		VideoID:      "abc123",
		Language:     "English",
		LanguageCode: "en",
		Lines:        []yt_transcript_models.TranscriptLine{{Text: "Hello", Start: 0, Duration: 1}},
	},
}

func TestNewClient_Defaults(t *testing.T) {
	client := NewClient()
	assert.Equal(t, 30, client.Timeout)
	assert.NotNil(t, client.Formatter)
	assert.NotNil(t, client.logger)
}

func TestWithTimeout(t *testing.T) {
	client := NewClient(WithTimeout(15))
	assert.Equal(t, 15, client.Timeout)
}

func TestWithFormatter(t *testing.T) {
	f := yt_transcript_formatters.NewTextFormatter()
	client := NewClient(WithFormatter(f))
	assert.Equal(t, f, client.Formatter)
}

func TestWithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewClient(WithLogger(logger))
	assert.Equal(t, logger, client.logger)
}

func TestGetTranscripts_DelegatesToService(t *testing.T) {
	svc := &mockService{transcripts: sampleTranscripts}
	client := NewClient()
	client.transcriptService = svc

	result, err := client.GetTranscripts(context.Background(), "abc123", []string{"en"})

	assert.NoError(t, err)
	assert.Equal(t, sampleTranscripts, result)
}

func TestGetTranscripts_PropagatesServiceError(t *testing.T) {
	svc := &mockService{err: errors.New("service failed")}
	client := NewClient()
	client.transcriptService = svc

	_, err := client.GetTranscripts(context.Background(), "abc123", []string{"en"})
	assert.EqualError(t, err, "service failed")
}

func TestGetTranscripts_AppliesTimeout(t *testing.T) {
	svc := &mockService{transcripts: sampleTranscripts}
	client := NewClient(WithTimeout(15))
	client.transcriptService = svc

	client.GetTranscripts(context.Background(), "abc123", []string{"en"})

	deadline, ok := svc.capturedCtx.Deadline()
	assert.True(t, ok, "context should have a deadline")
	assert.WithinDuration(t, time.Now().Add(15*time.Second), deadline, 2*time.Second)
}

func TestGetTranscripts_InjectsLogger(t *testing.T) {
	svc := &mockService{transcripts: sampleTranscripts}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewClient(WithLogger(logger))
	client.transcriptService = svc

	client.GetTranscripts(context.Background(), "abc123", []string{"en"})

	assert.Equal(t, logger, logging.FromContext(svc.capturedCtx))
}

func TestGetFormattedTranscripts_FormatsOutput(t *testing.T) {
	svc := &mockService{transcripts: sampleTranscripts}
	client := NewClient(WithFormatter(yt_transcript_formatters.NewJSONFormatter()))
	client.transcriptService = svc

	out, err := client.GetFormattedTranscripts(context.Background(), "abc123", []string{"en"}, false)

	assert.NoError(t, err)
	assert.JSONEq(t, `[{"language_code":"en","transcripts":[{"text":"Hello","duration":1}]}]`, out)
}

func TestGetFormattedTranscripts_EmptyResultError(t *testing.T) {
	svc := &mockService{transcripts: []yt_transcript_models.Transcript{}}
	client := NewClient()
	client.transcriptService = svc

	_, err := client.GetFormattedTranscripts(context.Background(), "abc123", []string{"en"}, false)
	assert.EqualError(t, err, "no transcripts found")
}

func TestGetFormattedTranscripts_PropagatesServiceError(t *testing.T) {
	svc := &mockService{err: errors.New("service failed")}
	client := NewClient()
	client.transcriptService = svc

	_, err := client.GetFormattedTranscripts(context.Background(), "abc123", []string{"en"}, false)
	assert.EqualError(t, err, "service failed")
}
