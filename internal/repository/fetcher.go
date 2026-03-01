package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

var videoBaseUrl = "https://www.youtube.com/watch?v=%s"

const INNERTUBE_API_URL = "https://www.youtube.com/youtubei/v1/player?key=%s"

var INNERTUBE_CONTEXT = map[string]interface{}{
	"client": map[string]interface{}{
		"clientName":    "ANDROID",
		"clientVersion": "20.10.38",
	},
}

var (
	consentURLRegex   = regexp.MustCompile(`https://consent\.youtube\.com/s`)
	consentValueRegex = regexp.MustCompile(`name="v" value="(.*?)"`)
)

type HTMLFetcherType interface {
	Fetch(url string, cookie *http.Cookie) ([]byte, error)
	FetchVideo(videoID string) ([]byte, error)
	FetchInnertubeData(ctx context.Context, videoID string, apiKey string, cookie *http.Cookie) (map[string]interface{}, error)
	FetchWithContext(ctx context.Context, url string, cookie *http.Cookie) ([]byte, error)
}

// Shared HTTP client with optimized connection pooling
var sharedHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	},
}

type HTMLFetcher struct{}

func NewHTMLFetcher() *HTMLFetcher {
	return &HTMLFetcher{}
}

func (f *HTMLFetcher) Fetch(url string, cookie *http.Cookie) ([]byte, error) {
	return f.FetchWithContext(context.Background(), url, cookie)
}

func (f *HTMLFetcher) FetchWithContext(ctx context.Context, url string, cookie *http.Cookie) ([]byte, error) {
	var lastErr error

	for range 3 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept-Language", "en-US")
		if cookie != nil {
			req.AddCookie(cookie)
		}

		resp, err := sharedHTTPClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("non-OK status code: %d", resp.StatusCode)
			time.Sleep(2 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}

		if len(body) > 0 {
			return body, nil
		}

		lastErr = fmt.Errorf("empty response body")
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("failed to fetch after retries: %w", lastErr)
}

func (f *HTMLFetcher) FetchVideo(videoID string) ([]byte, error) {
	videoURL := fmt.Sprintf(videoBaseUrl, videoID)

	body, err := f.Fetch(videoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch video page: %w", err)
	}

	if consentRequired(body) {
		cookie, err := f.createConsentCookie(videoURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create consent cookie: %w", err)
		}

		body, err = f.Fetch(videoURL, cookie)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch video page after setting consent: %w", err)
		}
	}

	return body, nil
}

func (f *HTMLFetcher) createConsentCookie(videoURL string) (*http.Cookie, error) {
	html, err := f.Fetch(videoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTML to extract consent value: %w", err)
	}

	match := consentValueRegex.FindSubmatch(html)
	if len(match) < 2 {
		return nil, fmt.Errorf("failed to find consent value in HTML")
	}

	cookie := &http.Cookie{
		Name:   "CONSENT",
		Value:  "YES+" + string(match[1]),
		Domain: ".youtube.com",
	}
	return cookie, nil
}

func consentRequired(body []byte) bool {
	return consentURLRegex.Match(body)
}

func (f *HTMLFetcher) FetchInnertubeData(ctx context.Context, videoID string, apiKey string, cookie *http.Cookie) (map[string]interface{}, error) {
	url := fmt.Sprintf(INNERTUBE_API_URL, apiKey)

	payload := map[string]interface{}{
		"context": INNERTUBE_CONTEXT,
		"videoId": videoID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if cookie != nil {
		req.AddCookie(cookie)
	}

	resp, err := sharedHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if consentRequired(body) && cookie == nil {
		cookie, err := f.createConsentCookie(videoID)
		if err != nil {
			return nil, fmt.Errorf("failed to create consent cookie: %w", err)
		}

		responseData, err := f.FetchInnertubeData(ctx, videoID, apiKey, cookie)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch video page after setting consent: %w", err)
		}
		return responseData, nil
	}

	var responseData map[string]interface{}
	if err = json.Unmarshal(body, &responseData); err != nil {
		return nil, fmt.Errorf("failed to decode response JSON: %w", err)
	}

	return responseData, nil
}
