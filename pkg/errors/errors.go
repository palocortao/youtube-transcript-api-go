package errors

import "fmt"

// NoTranscriptFoundError is returned when none of the requested languages have a transcript.
type NoTranscriptFoundError struct {
	Languages []string // requested language codes
	Available []string // actually available language codes
}

func (e *NoTranscriptFoundError) Error() string {
	return fmt.Sprintf("no transcript found for %v; available: %v", e.Languages, e.Available)
}

// TranscriptsDisabledError is returned when a video has no captions at all.
type TranscriptsDisabledError struct {
	VideoID string
}

func (e *TranscriptsDisabledError) Error() string {
	return fmt.Sprintf("transcripts are disabled for video %q", e.VideoID)
}
