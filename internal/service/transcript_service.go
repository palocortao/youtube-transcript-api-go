package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/palocortao/youtube-transcript-api-go/internal/logging"
	"github.com/palocortao/youtube-transcript-api-go/internal/repository"
	ytErrors "github.com/palocortao/youtube-transcript-api-go/pkg/errors"
	"github.com/palocortao/youtube-transcript-api-go/pkg/yt_transcript_models"
	"golang.org/x/net/html"
)

// errCaptionsNotFound is an internal sentinel returned by extractInnertubeVideoDetails
// when the response contains no captions data.
var errCaptionsNotFound = errors.New("captions not found in response")

type TranscriptService interface {
	GetTranscripts(ctx context.Context, videoID string, languages []string, preserveFormatting bool) ([]yt_transcript_models.Transcript, error)
}

type transcriptService struct {
	fetcher repository.HTMLFetcherType
}

type transcriptResult struct {
	transcript yt_transcript_models.Transcript
	err        error
}

func NewTranscriptService(fetcher repository.HTMLFetcherType) *transcriptService {
	return &transcriptService{fetcher: fetcher}
}

func (t transcriptService) GetTranscripts(ctx context.Context, videoID string, languages []string, preserveFormatting bool) ([]yt_transcript_models.Transcript, error) {
	videoID = sanitizeVideoId(videoID)

	logging.FromContext(ctx).DebugContext(ctx, "fetching transcripts", "videoID", videoID, "languages", languages)

	transcriptData, err := t.extractTranscriptList(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract list of transcripts: %w", err)
	}

	tracks, err := t.getTranscriptsForLanguage(languages, *transcriptData.Transcripts)
	if err != nil {
		return nil, fmt.Errorf("failed to get transcript: %w", err)
	}

	logging.FromContext(ctx).DebugContext(ctx, "found caption tracks", "videoID", videoID, "count", len(tracks))

	return t.processCaptionTracks(ctx, videoID, tracks, transcriptData.Title, preserveFormatting)
}

func (t *transcriptService) processCaptionTracks(ctx context.Context, videoID string, captionTracks []yt_transcript_models.CaptionTrack, title string, preserveFormatting bool) ([]yt_transcript_models.Transcript, error) {
	resultChan := make(chan transcriptResult, len(captionTracks))
	var wg sync.WaitGroup

	results := make([]yt_transcript_models.Transcript, 0, len(captionTracks))

	for _, transcript := range captionTracks {
		wg.Add(1)
		go func(tr yt_transcript_models.CaptionTrack) {
			defer wg.Done()

			logging.FromContext(ctx).DebugContext(ctx, "fetching caption track", "videoID", videoID, "language", tr.LanguageCode)

			isGenerated := tr.Kind != nil && *tr.Kind == "asr"

			lines, err := t.getTranscriptFromTrack(ctx, tr, preserveFormatting)
			if err != nil {
				resultChan <- transcriptResult{err: fmt.Errorf("error getting transcript from track: %w", err)}
				return
			}

			resultChan <- transcriptResult{transcript: yt_transcript_models.Transcript{
				VideoID:        videoID,
				VideoTitle:     title,
				Language:       tr.Name.SimpleText,
				LanguageCode:   tr.LanguageCode,
				IsGenerated:    isGenerated,
				IsTranslatable: tr.IsTranslatable,
				Lines:          lines,
			}}
		}(transcript)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if result.err != nil {
			return nil, result.err
		}
		results = append(results, result.transcript)
	}
	return results, nil
}

// Pre-compiled regex for API key extraction
var innerTubeApiKeyRegex = regexp.MustCompile(`"INNERTUBE_API_KEY":\s*"([a-zA-Z0-9_-]+)"`)

func extractInnerTubeApiKey(htmlContent string) string {
	match := innerTubeApiKeyRegex.FindStringSubmatch(htmlContent)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func extractInnertubeVideoDetails(data map[string]interface{}) (*yt_transcript_models.InnertubeData, error) {
	captions, ok := data["captions"].(map[string]interface{})
	if !ok {
		return nil, errCaptionsNotFound
	}

	renderer, ok := captions["playerCaptionsTracklistRenderer"].(map[string]interface{})
	if !ok {
		return nil, errCaptionsNotFound
	}

	var captionTracks []yt_transcript_models.CaptionTrack
	if tracks, ok := renderer["captionTracks"].([]interface{}); ok {
		captionTracks = make([]yt_transcript_models.CaptionTrack, 0, len(tracks))
		for _, track := range tracks {
			if trackMap, ok := track.(map[string]interface{}); ok {
				captionTrack := yt_transcript_models.CaptionTrack{}

				if baseUrl, ok := trackMap["baseUrl"].(string); ok {
					captionTrack.BaseUrl = baseUrl
				}

				if langCode, ok := trackMap["languageCode"].(string); ok {
					captionTrack.LanguageCode = langCode
				}

				if name, ok := trackMap["name"].(map[string]interface{}); ok {
					if simpleText, ok := name["simpleText"].(string); ok {
						captionTrack.Name = yt_transcript_models.LanguageName{SimpleText: simpleText}
					}
				}

				if kind, ok := trackMap["kind"].(string); ok {
					captionTrack.Kind = &kind
				}

				if isTranslatable, ok := trackMap["isTranslatable"].(bool); ok {
					captionTrack.IsTranslatable = isTranslatable
				}

				captionTracks = append(captionTracks, captionTrack)
			}
		}
	}

	var translationLanguages *[]yt_transcript_models.LanguageData
	if transLangs, ok := renderer["translationLanguages"].([]interface{}); ok {
		langs := make([]yt_transcript_models.LanguageData, 0, len(transLangs))
		for _, lang := range transLangs {
			if langMap, ok := lang.(map[string]interface{}); ok {
				langData := yt_transcript_models.LanguageData{}

				if langCode, ok := langMap["languageCode"].(string); ok {
					langData.LanguageCode = langCode
				}

				if langName, ok := langMap["languageName"].(map[string]interface{}); ok {
					if simpleText, ok := langName["simpleText"].(string); ok {
						langData.Language = yt_transcript_models.LanguageName{SimpleText: simpleText}
					}
				}

				langs = append(langs, langData)
			}
		}
		if len(langs) > 0 {
			translationLanguages = &langs
		}
	}

	transcriptData := &yt_transcript_models.TranscriptData{
		CaptionTracks:        captionTracks,
		TranslationLanguages: translationLanguages,
	}

	return &yt_transcript_models.InnertubeData{
		Captions: yt_transcript_models.CaptionsDetails{
			PlayerCaptionsTracklistRenderer: transcriptData,
		},
	}, nil
}

func extractTitle(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}

	var title string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil {
				title = n.FirstChild.Data
				return
			}
		}
		for c := n.FirstChild; c != nil && title == ""; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)
	return title
}

func (t *transcriptService) extractTranscriptList(ctx context.Context, videoID string) (*yt_transcript_models.VideoTranscriptData, error) {
	logging.FromContext(ctx).DebugContext(ctx, "fetching video page", "videoID", videoID)

	htmlBytes, err := t.fetcher.FetchVideo(videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch video page: %w", err)
	}

	body := string(htmlBytes)
	title := extractTitle(body)

	logging.FromContext(ctx).DebugContext(ctx, "fetched video page", "videoID", videoID, "title", title)

	innertubeAPIKey := extractInnerTubeApiKey(body)

	logging.FromContext(ctx).DebugContext(ctx, "fetching innertube data", "videoID", videoID)

	innertubeData, err := t.fetcher.FetchInnertubeData(ctx, videoID, innertubeAPIKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch innertube data: %w", err)
	}

	videoDetails, err := extractInnertubeVideoDetails(innertubeData)
	if errors.Is(err, errCaptionsNotFound) {
		return nil, &ytErrors.TranscriptsDisabledError{VideoID: videoID}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to extract video details: %w", err)
	}

	transcripts := videoDetails.Captions.PlayerCaptionsTracklistRenderer

	return &yt_transcript_models.VideoTranscriptData{Transcripts: transcripts, Title: title}, nil
}

func (s transcriptService) getTranscriptsForLanguage(languages []string, transcripts yt_transcript_models.TranscriptData) ([]yt_transcript_models.CaptionTrack, error) {
	if len(languages) == 0 {
		return transcripts.CaptionTracks, nil
	}

	captionTracks := make([]yt_transcript_models.CaptionTrack, 0, len(languages))

	for _, lang := range languages {
		for _, track := range transcripts.CaptionTracks {
			if track.LanguageCode == lang {
				captionTracks = append(captionTracks, track)
			}
		}
	}

	if len(captionTracks) == 0 {
		available := make([]string, len(transcripts.CaptionTracks))
		for i, t := range transcripts.CaptionTracks {
			available[i] = t.LanguageCode
		}
		return nil, &ytErrors.NoTranscriptFoundError{Languages: languages, Available: available}
	}

	return captionTracks, nil
}

func (s transcriptService) getTranscriptFromTrack(ctx context.Context, track yt_transcript_models.CaptionTrack, preserveFormatting bool) ([]yt_transcript_models.TranscriptLine, error) {
	trackURL := strings.Replace(track.BaseUrl, "&fmt=srv3", "", -1)
	body, err := s.fetcher.FetchWithContext(ctx, trackURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transcript: %w", err)
	}

	parser := repository.NewTranscriptParser(preserveFormatting)

	transcript, err := parser.Parse(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse transcript: %w", err)
	}
	return transcript, nil
}

func sanitizeVideoId(videoID string) string {
	if strings.HasPrefix(videoID, "http://") || strings.HasPrefix(videoID, "https://") || strings.HasPrefix(videoID, "www.") {
		if strings.Contains(videoID, "youtube.com") {
			u, err := url.Parse(videoID)
			if err != nil {
				return videoID
			}
			return u.Query().Get("v")
		} else if strings.Contains(videoID, "youtu.be") {
			u, err := url.Parse(videoID)
			if err != nil {
				return videoID
			}
			return strings.TrimPrefix(u.Path, "/")
		}
	}
	return videoID
}
