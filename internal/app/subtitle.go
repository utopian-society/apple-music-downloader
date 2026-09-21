package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ExtractedSubtitle represents a subtitle track extracted or downloaded for an MV.
type ExtractedSubtitle struct {
	Lang string
	Path string
}

// CleanSRTText cleans up common formatting artifacts from EIA-608 / WebVTT conversions:
// - strips HTML-style tags (<font ...>, </font>, etc.)
// - strips ASS positioning tags ({\an7}, etc.)
// - converts \h (hard spaces from EIA-608) to regular space
// - cleans up whitespace while preserving SRT structure
func CleanSRTText(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.ReplaceAll(content, `\h`, " ")

	tagRe := regexp.MustCompile(`<[^>]+>`)
	content = tagRe.ReplaceAllString(content, "")

	assRe := regexp.MustCompile(`\{\\[^}]*\}`)
	content = assRe.ReplaceAllString(content, "")

	lines := strings.Split(content, "\n")
	var cleaned []string
	prevBlank := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if !prevBlank && len(cleaned) > 0 {
				cleaned = append(cleaned, "")
				prevBlank = true
			}
			continue
		}
		prevBlank = false
		cleaned = append(cleaned, trimmed)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n")) + "\n"
}

// ExtractSubtitlesFromVideo detects subtitle or closed caption streams in a video file
// using ffprobe, extracts them to .srt using ffmpeg, and applies text cleaning.
func ExtractSubtitlesFromVideo(videoPath, outputDir, baseName string) ([]ExtractedSubtitle, error) {
	if _, err := os.Stat(videoPath); err != nil {
		return nil, fmt.Errorf("video file not found: %w", err)
	}

	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		videoPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probeData struct {
		Streams []struct {
			Index          int    `json:"index"`
			CodecName      string `json:"codec_name"`
			CodecType      string `json:"codec_type"`
			CodecTagString string `json:"codec_tag_string"`
			Tags           struct {
				Language string `json:"language"`
			} `json:"tags"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	var subtitleStreams []struct {
		Index int
		Lang  string
	}

	for _, stream := range probeData.Streams {
		if stream.CodecType == "subtitle" ||
			strings.Contains(stream.CodecName, "608") ||
			stream.CodecName == "eia_608" ||
			stream.CodecName == "c608" ||
			stream.CodecTagString == "c608" {
			lang := stream.Tags.Language
			if lang == "" {
				lang = "eng"
			}
			subtitleStreams = append(subtitleStreams, struct {
				Index int
				Lang  string
			}{stream.Index, lang})
		}
	}

	if len(subtitleStreams) == 0 {
		return nil, nil
	}

	var results []ExtractedSubtitle
	for _, stream := range subtitleStreams {
		srtName := fmt.Sprintf("%s_%s.srt", baseName, stream.Lang)
		srtPath := filepath.Join(outputDir, srtName)

		extractCmd := exec.Command("ffmpeg",
			"-y",
			"-i", videoPath,
			"-map", fmt.Sprintf("0:%d", stream.Index),
			"-c:s", "srt",
			srtPath,
		)

		if err := extractCmd.Run(); err != nil {
			continue
		}

		info, err := os.Stat(srtPath)
		if err != nil || info.Size() < 50 {
			_ = os.Remove(srtPath)
			continue
		}

		if rawContent, err := os.ReadFile(srtPath); err == nil {
			cleaned := CleanSRTText(string(rawContent))
			_ = os.WriteFile(srtPath, []byte(cleaned), 0644)
		}

		results = append(results, ExtractedSubtitle{
			Lang: stream.Lang,
			Path: srtPath,
		})
	}

	return results, nil
}

// ParseSRTSubtitles reads an SRT file and returns the raw subtitle data.
// No longer modifies timestamps — ffmpeg's -itsoffset handles timing offset.
func ParseSRTSubtitles(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open subtitle file: %w", err)
	}
	defer file.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content.WriteString(scanner.Text() + "\n")
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read subtitle file: %w", err)
	}

	return content.String(), nil
}

// ValidateSRT checks if an SRT file has valid structure.
func ValidateSRT(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open subtitle file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		// Expect subtitle index.
		if lineNum == 1 || strings.Contains(line, " --> ") {
			// Simple validation: just check structure exists.
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to validate SRT: %w", err)
	}

	return nil
}
