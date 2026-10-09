package repository

import (
	"testing"

	"github.com/palocortao/youtube-transcript-api-go/pkg/yt_transcript_models"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name               string
		xml                string
		preserveFormatting bool
		expected           []yt_transcript_models.TranscriptLine
		expectError        bool
	}{
		{
			name: "basic transcript",
			xml:  `<?xml version="1.0" encoding="utf-8"?><transcript><text start="1.5" dur="2.0">Hello world</text></transcript>`,
			expected: []yt_transcript_models.TranscriptLine{
				{Text: "Hello world", Start: 1.5, Duration: 2.0},
			},
		},
		{
			name: "multiple lines",
			xml: `<?xml version="1.0" encoding="utf-8"?><transcript>` +
				`<text start="0" dur="1">Line one</text>` +
				`<text start="1" dur="2">Line two</text>` +
				`</transcript>`,
			expected: []yt_transcript_models.TranscriptLine{
				{Text: "Line one", Start: 0, Duration: 1},
				{Text: "Line two", Start: 1, Duration: 2},
			},
		},
		{
			name: "HTML entities unescaped",
			xml:  `<?xml version="1.0" encoding="utf-8"?><transcript><text start="0" dur="1">Hello &amp;amp; world</text></transcript>`,
			expected: []yt_transcript_models.TranscriptLine{
				{Text: "Hello & world", Start: 0, Duration: 1},
			},
		},
		{
			name: "HTML tags stripped from content",
			xml:  `<?xml version="1.0" encoding="utf-8"?><transcript><text start="0" dur="1">&lt;font color="#fff"&gt;Hello&lt;/font&gt;</text></transcript>`,
			expected: []yt_transcript_models.TranscriptLine{
				{Text: "Hello", Start: 0, Duration: 1},
			},
		},
		{
			name: "missing start attribute defaults to zero",
			xml:  `<?xml version="1.0" encoding="utf-8"?><transcript><text dur="1.5">Hello</text></transcript>`,
			expected: []yt_transcript_models.TranscriptLine{
				{Text: "Hello", Start: 0, Duration: 1.5},
			},
		},
		{
			name: "missing duration attribute defaults to zero",
			xml:  `<?xml version="1.0" encoding="utf-8"?><transcript><text start="2.5">Hello</text></transcript>`,
			expected: []yt_transcript_models.TranscriptLine{
				{Text: "Hello", Start: 2.5, Duration: 0},
			},
		},
		{
			name:     "empty transcript returns nil",
			xml:      `<?xml version="1.0" encoding="utf-8"?><transcript></transcript>`,
			expected: nil,
		},
		{
			name:        "invalid XML returns error",
			xml:         `this is not xml {{{}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewTranscriptParser(tt.preserveFormatting)
			result, err := parser.Parse(tt.xml)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
