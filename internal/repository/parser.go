package repository

import (
	"encoding/xml"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/horiagug/youtube-transcript-api-go/pkg/yt_transcript_models"
)

type transcriptParser struct {
	preserveFormatting bool
}

var formattingTags = []string{
	"strong", "em", "b", "i", "mark", "small", "del", "ins", "sub", "sup",
}

var stripHTMLRegex = regexp.MustCompile(`(?i)<[^>]*>`)

func NewTranscriptParser(preserveFormatting bool) *transcriptParser {
	return &transcriptParser{preserveFormatting: preserveFormatting}
}

func cleanHTML(text string, preserveFormatting bool) string {
	cleaned := stripHTMLRegex.ReplaceAllString(text, "")

	if preserveFormatting {
		for _, tag := range formattingTags {
			cleaned = strings.ReplaceAll(cleaned, "&lt;"+tag+"&gt;", "<"+tag+">")
			cleaned = strings.ReplaceAll(cleaned, "&lt;/"+tag+"&gt;", "</"+tag+">")
		}
	}

	return cleaned
}

func (p *transcriptParser) Parse(plainData string) ([]yt_transcript_models.TranscriptLine, error) {
	type XMLTranscript struct {
		XMLName xml.Name `xml:"transcript"`
		Texts   []struct {
			Text     string `xml:",chardata"`
			Start    string `xml:"start,attr"`
			Duration string `xml:"dur,attr"`
		} `xml:"text"`
	}

	var parsedXML XMLTranscript
	err := xml.Unmarshal([]byte(plainData), &parsedXML)
	if err != nil {
		return nil, err
	}

	var results []yt_transcript_models.TranscriptLine
	for _, entry := range parsedXML.Texts {
		text := cleanHTML(entry.Text, p.preserveFormatting)
		text = html.UnescapeString(text)

		start, err := strconv.ParseFloat(entry.Start, 64)
		if err != nil {
			start = 0.0
		}

		duration, err := strconv.ParseFloat(entry.Duration, 64)
		if err != nil {
			duration = 0.0
		}

		results = append(results, yt_transcript_models.TranscriptLine{
			Text:     text,
			Start:    start,
			Duration: duration,
		})
	}
	return results, nil
}
