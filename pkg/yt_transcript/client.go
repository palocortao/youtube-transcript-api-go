package yt_transcript

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/palocortao/youtube-transcript-api-go/internal/logging"
	"github.com/palocortao/youtube-transcript-api-go/internal/repository"
	"github.com/palocortao/youtube-transcript-api-go/internal/service"
	"github.com/palocortao/youtube-transcript-api-go/pkg/yt_transcript_formatters"
	"github.com/palocortao/youtube-transcript-api-go/pkg/yt_transcript_models"
)

type YtTranscriptClient struct {
	transcriptService service.TranscriptService
	Timeout           int
	Formatter         yt_transcript_formatters.Formatter
	logger            *slog.Logger
}

func NewClient(options ...Option) *YtTranscriptClient {
	formatter := yt_transcript_formatters.NewJSONFormatter()
	formatter.Configure(yt_transcript_formatters.WithPrettyPrint(true))

	client := &YtTranscriptClient{
		Timeout:   30,
		Formatter: formatter,
		logger:    slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	for _, opt := range options {
		opt(client)
	}

	if client.transcriptService == nil {
		fetcher := repository.NewHTMLFetcher()
		client.transcriptService = service.NewTranscriptService(fetcher)
	}

	return client
}

func (c *YtTranscriptClient) GetFormattedTranscripts(ctx context.Context, videoID string, languages []string, preserveFormatting bool) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.Timeout)*time.Second)
	defer cancel()

	ctx = logging.WithLogger(ctx, c.logger)

	transcripts, err := c.transcriptService.GetTranscripts(ctx, videoID, languages, preserveFormatting)
	if err != nil {
		return "", err
	}

	if len(transcripts) == 0 {
		return "", fmt.Errorf("no transcripts found")
	}

	return c.Formatter.Format(transcripts)
}

func (c *YtTranscriptClient) GetTranscripts(ctx context.Context, videoID string, languages []string) ([]yt_transcript_models.Transcript, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.Timeout)*time.Second)
	defer cancel()

	ctx = logging.WithLogger(ctx, c.logger)

	return c.transcriptService.GetTranscripts(ctx, videoID, languages, true)
}
