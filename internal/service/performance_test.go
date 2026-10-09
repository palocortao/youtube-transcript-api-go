package service

import (
	"context"
	"testing"
	"time"

	"github.com/palocortao/youtube-transcript-api-go/internal/repository/fixtures"
	"github.com/palocortao/youtube-transcript-api-go/pkg/yt_transcript_models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestContextTimeoutRespected(t *testing.T) {
	fetcher := &fixtures.MockHTMLFetcher{}
	service := NewTranscriptService(fetcher)

	fetcher.On("FetchVideo", mock.AnythingOfType("string")).Return([]byte(`<title>Test Video</title>"INNERTUBE_API_KEY":"test_key"`), nil)

	mockInnertubeData := map[string]interface{}{
		"captions": map[string]interface{}{
			"playerCaptionsTracklistRenderer": map[string]interface{}{
				"captionTracks": []interface{}{
					map[string]interface{}{
						"baseUrl":        "http://example.com/transcript",
						"name":           map[string]interface{}{"simpleText": "English"},
						"languageCode":   "en",
						"kind":           "asr",
						"isTranslatable": true,
					},
				},
			},
		},
	}
	fetcher.On("FetchInnertubeData", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).Return(mockInnertubeData, nil)

	// Mock a slow transcript fetch — blocks until context is cancelled
	fetcher.On("FetchWithContext", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
			t.Error("Context timeout was not respected")
		}
	}).Return([]byte{}, context.DeadlineExceeded)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := service.GetTranscripts(ctx, "test123", []string{"en"}, false)
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Less(t, elapsed, 500*time.Millisecond, "Context timeout should have been respected")
}

func TestGetTranscriptsForLanguage(t *testing.T) {
	service := transcriptService{}

	transcripts := yt_transcript_models.TranscriptData{
		CaptionTracks: []yt_transcript_models.CaptionTrack{
			{LanguageCode: "en", Name: yt_transcript_models.LanguageName{SimpleText: "English"}},
			{LanguageCode: "es", Name: yt_transcript_models.LanguageName{SimpleText: "Spanish"}},
		},
	}

	t.Run("Empty languages returns all tracks", func(t *testing.T) {
		result, err := service.getTranscriptsForLanguage([]string{}, transcripts)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("Specific language filters correctly", func(t *testing.T) {
		result, err := service.getTranscriptsForLanguage([]string{"en"}, transcripts)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "en", result[0].LanguageCode)
	})

	t.Run("Unknown language returns error", func(t *testing.T) {
		result, err := service.getTranscriptsForLanguage([]string{"fr"}, transcripts)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
