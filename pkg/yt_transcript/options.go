package yt_transcript

import (
	"log/slog"

	"github.com/palocortao/youtube-transcript-api-go/internal/repository"
	"github.com/palocortao/youtube-transcript-api-go/internal/service"
	"github.com/palocortao/youtube-transcript-api-go/pkg/yt_transcript_formatters"
)

type Option func(*YtTranscriptClient)

func WithCustomFetcher(fetcher repository.HTMLFetcherType) Option {
	return func(c *YtTranscriptClient) {
		c.transcriptService = service.NewTranscriptService(fetcher)
	}
}

func WithTimeout(seconds int) Option {
	return func(c *YtTranscriptClient) {
		c.Timeout = seconds
	}
}

func WithFormatter(formatter yt_transcript_formatters.Formatter) Option {
	return func(c *YtTranscriptClient) {
		c.Formatter = formatter
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(c *YtTranscriptClient) {
		c.logger = logger
	}
}
