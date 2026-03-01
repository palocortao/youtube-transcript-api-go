package yt_transcript_formatters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/horiagug/youtube-transcript-api-go/pkg/yt_transcript_models"
)

var sampleTranscript = yt_transcript_models.Transcript{
	VideoID:      "abc123",
	VideoTitle:   "Test Video",
	Language:     "English",
	LanguageCode: "en",
	Lines: []yt_transcript_models.TranscriptLine{
		{Text: "Hello world", Start: 1.0, Duration: 1.5},
		{Text: "Goodbye world", Start: 2.5, Duration: 2.0},
	},
}

var spanishTranscript = yt_transcript_models.Transcript{
	VideoID:      "abc123",
	Language:     "Spanish",
	LanguageCode: "es",
	Lines: []yt_transcript_models.TranscriptLine{
		{Text: "Hola mundo", Start: 1.0, Duration: 1.5},
	},
}

// --- JSONFormatter ---

func TestJSONFormatter_Default(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format([]yt_transcript_models.Transcript{sampleTranscript})
	assert.NoError(t, err)
	assert.JSONEq(t, `[{"language_code":"en","transcripts":[{"text":"Hello world","start":1,"duration":1.5},{"text":"Goodbye world","start":2.5,"duration":2}]}]`, out)
}

func TestJSONFormatter_WithoutTimestamps(t *testing.T) {
	f := NewJSONFormatter(WithTimestamps(false))
	out, err := f.Format([]yt_transcript_models.Transcript{sampleTranscript})
	assert.NoError(t, err)
	assert.JSONEq(t, `[{"language_code":"en","transcripts":[{"text":"Hello world"},{"text":"Goodbye world"}]}]`, out)
}

func TestJSONFormatter_WithoutLanguageCode(t *testing.T) {
	f := NewJSONFormatter(WithLanguageCode(false))
	out, err := f.Format([]yt_transcript_models.Transcript{sampleTranscript})
	assert.NoError(t, err)
	assert.JSONEq(t, `[{"language_code":null,"transcripts":[{"text":"Hello world","start":1,"duration":1.5},{"text":"Goodbye world","start":2.5,"duration":2}]}]`, out)
}

func TestJSONFormatter_PrettyPrint(t *testing.T) {
	f := NewJSONFormatter()
	f.Configure(WithPrettyPrint(true))
	out, err := f.Format([]yt_transcript_models.Transcript{sampleTranscript})
	assert.NoError(t, err)
	// Pretty-printed output contains newlines and indentation
	assert.Contains(t, out, "\n")
	assert.Contains(t, out, "  ")
	// Content is still semantically correct
	assert.JSONEq(t, `[{"language_code":"en","transcripts":[{"text":"Hello world","start":1,"duration":1.5},{"text":"Goodbye world","start":2.5,"duration":2}]}]`, out)
}

func TestJSONFormatter_MultipleTranscripts(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format([]yt_transcript_models.Transcript{sampleTranscript, spanishTranscript})
	assert.NoError(t, err)
	assert.JSONEq(t, `[
		{"language_code":"en","transcripts":[{"text":"Hello world","start":1,"duration":1.5},{"text":"Goodbye world","start":2.5,"duration":2}]},
		{"language_code":"es","transcripts":[{"text":"Hola mundo","start":1,"duration":1.5}]}
	]`, out)
}

func TestJSONFormatter_EmptyInput(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format([]yt_transcript_models.Transcript{})
	assert.NoError(t, err)
	assert.JSONEq(t, `[]`, out)
}

// --- TextFormatter ---

func TestTextFormatter_Default(t *testing.T) {
	f := NewTextFormatter()
	out, err := f.Format([]yt_transcript_models.Transcript{sampleTranscript})
	assert.NoError(t, err)
	assert.Equal(t, "Language: English\n1.000000: Hello world\n2.500000: Goodbye world\n", out)
}

func TestTextFormatter_WithoutTimestamps(t *testing.T) {
	f := NewTextFormatter(WithTimestamps(false))
	out, err := f.Format([]yt_transcript_models.Transcript{sampleTranscript})
	assert.NoError(t, err)
	assert.Equal(t, "Language: English\nHello world\nGoodbye world\n", out)
}

func TestTextFormatter_WithoutLanguageCode(t *testing.T) {
	f := NewTextFormatter(WithLanguageCode(false))
	out, err := f.Format([]yt_transcript_models.Transcript{sampleTranscript})
	assert.NoError(t, err)
	assert.Equal(t, "1.000000: Hello world\n2.500000: Goodbye world\n", out)
}

func TestTextFormatter_MultipleTranscripts_HasSeparator(t *testing.T) {
	f := NewTextFormatter()
	out, err := f.Format([]yt_transcript_models.Transcript{sampleTranscript, spanishTranscript})
	assert.NoError(t, err)
	assert.Contains(t, out, "Language: English")
	assert.Contains(t, out, "Language: Spanish")
	// blank line separator between transcripts
	assert.Contains(t, out, "Goodbye world\n\nLanguage: Spanish")
}

func TestTextFormatter_LanguageCodeFallback(t *testing.T) {
	// Language is empty, falls back to LanguageCode
	noNameTranscript := yt_transcript_models.Transcript{
		Language:     "",
		LanguageCode: "en",
		Lines:        []yt_transcript_models.TranscriptLine{{Text: "Hello", Start: 0, Duration: 1}},
	}
	f := NewTextFormatter()
	out, err := f.Format([]yt_transcript_models.Transcript{noNameTranscript})
	assert.NoError(t, err)
	assert.Contains(t, out, "Language: en")
}
