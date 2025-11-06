package subtitle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
)

// SubtitleTrack represents a subtitle track with language information
type SubtitleTrack struct {
	Language     string
	LanguageCode string
	URL          string
	Format       string
}

// MusicVideoSubtitles represents the subtitle data from Apple Music API
type MusicVideoSubtitles struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Ttml              string `json:"ttml"`
			TtmlLocalizations string `json:"ttmlLocalizations"`
			PlayParams        struct {
				ID          string `json:"id"`
				Kind        string `json:"kind"`
				CatalogID   string `json:"catalogId"`
				DisplayType int    `json:"displayType"`
			} `json:"playParams"`
		} `json:"attributes"`
	} `json:"data"`
}

// SRTSubtitle represents a single subtitle entry in SRT format
type SRTSubtitle struct {
	Index     int
	StartTime time.Duration
	EndTime   time.Duration
	Text      string
}

// Get fetches subtitles for a music video and converts to SRT format
func Get(storefront, musicVideoID, language, format, token, mediaUserToken string) (string, error) {
	if len(mediaUserToken) < 50 {
		return "", errors.New("MediaUserToken not set")
	}

	ttml, err := getMusicVideoSubtitles(musicVideoID, storefront, token, mediaUserToken, language)
	if err != nil {
		return "", err
	}

	if ttml == "" {
		return "", errors.New("no subtitles available for this music video")
	}

	if format == "ttml" {
		return ttml, nil
	}

	// Convert TTML to SRT
	srt, err := TTMLToSRT(ttml)
	if err != nil {
		return "", err
	}

	return srt, nil
}

// getMusicVideoSubtitles fetches subtitle data from Apple Music API
func getMusicVideoSubtitles(musicVideoID, storefront, token, userToken, language string) (string, error) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("https://amp-api.music.apple.com/v1/catalog/%s/music-videos/%s/subtitles?l=%s",
			storefront, musicVideoID, language), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Origin", "https://music.apple.com")
	req.Header.Set("Referer", "https://music.apple.com/")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	cookie := http.Cookie{Name: "media-user-token", Value: userToken}
	req.AddCookie(&cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch subtitles: %s", resp.Status)
	}

	obj := new(MusicVideoSubtitles)
	err = json.NewDecoder(resp.Body).Decode(&obj)
	if err != nil {
		return "", err
	}

	if obj.Data != nil && len(obj.Data) > 0 {
		if len(obj.Data[0].Attributes.Ttml) > 0 {
			return obj.Data[0].Attributes.Ttml, nil
		}
		return obj.Data[0].Attributes.TtmlLocalizations, nil
	}

	return "", errors.New("no subtitle data found")
}

// ExtractSubtitlesFromM3U8 extracts subtitle tracks from M3U8 playlist
func ExtractSubtitlesFromM3U8(m3u8URL string) ([]SubtitleTrack, error) {
	resp, err := http.Get(m3u8URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch m3u8: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	content := string(body)
	var subtitleTracks []SubtitleTrack

	// Parse M3U8 for subtitle tracks
	// Looking for EXT-X-MEDIA tags with TYPE=SUBTITLES
	lines := strings.Split(content, "\n")
	baseURL, _ := url.Parse(m3u8URL)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#EXT-X-MEDIA:") && strings.Contains(line, "TYPE=SUBTITLES") {
			track := SubtitleTrack{}

			// Extract NAME
			if nameMatch := regexp.MustCompile(`NAME="([^"]+)"`).FindStringSubmatch(line); len(nameMatch) > 1 {
				track.Language = nameMatch[1]
			}

			// Extract LANGUAGE
			if langMatch := regexp.MustCompile(`LANGUAGE="([^"]+)"`).FindStringSubmatch(line); len(langMatch) > 1 {
				track.LanguageCode = langMatch[1]
			}

			// Extract URI
			if uriMatch := regexp.MustCompile(`URI="([^"]+)"`).FindStringSubmatch(line); len(uriMatch) > 1 {
				subtitleURL := uriMatch[1]
				// Convert relative URL to absolute
				if !strings.HasPrefix(subtitleURL, "http") {
					parsedURL, err := baseURL.Parse(subtitleURL)
					if err == nil {
						subtitleURL = parsedURL.String()
					}
				}
				track.URL = subtitleURL
			}

			// Determine format from URL
			if strings.HasSuffix(track.URL, ".vtt") || strings.Contains(track.URL, ".vtt") {
				track.Format = "webvtt"
			} else if strings.HasSuffix(track.URL, ".ttml") || strings.Contains(track.URL, ".ttml") {
				track.Format = "ttml"
			} else if strings.HasSuffix(track.URL, ".srt") || strings.Contains(track.URL, ".srt") {
				track.Format = "srt"
			} else {
				track.Format = "unknown"
			}

			if track.URL != "" {
				subtitleTracks = append(subtitleTracks, track)
			}
		}

		// Also check for segments if next line is a URL
		if i+1 < len(lines) && strings.HasPrefix(line, "#EXT-X-MEDIA:") {
			nextLine := strings.TrimSpace(lines[i+1])
			if strings.HasPrefix(nextLine, "http") || strings.HasSuffix(nextLine, ".vtt") || strings.HasSuffix(nextLine, ".ttml") {
				// This is a segment URL
				continue
			}
		}
	}

	return subtitleTracks, nil
}

// DownloadSubtitle downloads subtitle content from URL
func DownloadSubtitle(subtitleURL string) (string, error) {
	resp, err := http.Get(subtitleURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download subtitle: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// extractTextFromTTMLElement recursively extracts text content from TTML element
// This properly handles nested elements like <span>, <br/>, etc.
func extractTextFromTTMLElement(elem *etree.Element) string {
	var textParts []string

	// Process all child nodes (text and elements)
	for _, child := range elem.Child {
		switch node := child.(type) {
		case *etree.CharData:
			// Direct text content
			text := node.Data
			// Remove escape sequences and special characters
			text = strings.ReplaceAll(text, "\\h", "")
			text = strings.ReplaceAll(text, "\\n", "\n")
			text = strings.ReplaceAll(text, "\\t", " ")
			text = strings.TrimSpace(text)
			if text != "" {
				textParts = append(textParts, text)
			}
		case *etree.Element:
			// Handle break elements
			if node.Tag == "br" {
				textParts = append(textParts, "\n")
			} else {
				// Recursively get text from nested elements
				text := extractTextFromTTMLElement(node)
				if text != "" {
					textParts = append(textParts, text)
				}
			}
		}
	}

	result := strings.Join(textParts, " ")

	// Remove any remaining escape sequences or special patterns
	result = regexp.MustCompile(`\\[a-z]`).ReplaceAllString(result, "")

	// Clean up multiple spaces and trim
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = strings.ReplaceAll(result, " \n ", "\n")
	result = strings.ReplaceAll(result, " \n", "\n")
	result = strings.ReplaceAll(result, "\n ", "\n")

	return strings.TrimSpace(result)
}

// TTMLToSRT converts TTML format to SRT format
func TTMLToSRT(ttml string) (string, error) {
	parsedTTML := etree.NewDocument()
	err := parsedTTML.ReadFromString(ttml)
	if err != nil {
		return "", err
	}

	var subtitles []SRTSubtitle
	index := 1

	// Find all p (paragraph) elements with timing information
	for _, p := range parsedTTML.FindElements("//p") {
		beginAttr := p.SelectAttr("begin")
		endAttr := p.SelectAttr("end")

		if beginAttr == nil || endAttr == nil {
			continue
		}

		startTime, err := parseTimeCode(beginAttr.Value)
		if err != nil {
			continue
		}

		endTime, err := parseTimeCode(endAttr.Value)
		if err != nil {
			continue
		}

		text := extractTextFromTTMLElement(p)
		if text == "" {
			continue
		}

		subtitles = append(subtitles, SRTSubtitle{
			Index:     index,
			StartTime: startTime,
			EndTime:   endTime,
			Text:      text,
		})
		index++
	}

	if len(subtitles) == 0 {
		return "", errors.New("no subtitle entries found in TTML")
	}

	// Sort by start time
	sort.Slice(subtitles, func(i, j int) bool {
		return subtitles[i].StartTime < subtitles[j].StartTime
	})

	// Convert to SRT format
	srt := convertToSRT(subtitles)

	// Apply final cleaning to remove any remaining formatting tags
	srt = string(removeFormattingTags([]byte(srt)))

	return srt, nil
}

// WebVTTToSRT converts WebVTT format to SRT format
func WebVTTToSRT(webvtt string) (string, error) {
	lines := strings.Split(webvtt, "\n")
	var subtitles []SRTSubtitle
	index := 1

	inCue := false
	var currentStart, currentEnd time.Duration
	var currentText []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip WEBVTT header and NOTE blocks
		if strings.HasPrefix(line, "WEBVTT") || strings.HasPrefix(line, "NOTE") {
			continue
		}

		// Check if this is a timestamp line (contains -->)
		if strings.Contains(line, "-->") {
			parts := strings.Split(line, "-->")
			if len(parts) == 2 {
				start, err := parseWebVTTTime(strings.TrimSpace(parts[0]))
				if err != nil {
					continue
				}
				end, err := parseWebVTTTime(strings.TrimSpace(parts[1]))
				if err != nil {
					continue
				}

				// Save previous subtitle if exists
				if inCue && len(currentText) > 0 {
					subtitles = append(subtitles, SRTSubtitle{
						Index:     index,
						StartTime: currentStart,
						EndTime:   currentEnd,
						Text:      strings.Join(currentText, "\n"),
					})
					index++
				}

				currentStart = start
				currentEnd = end
				currentText = []string{}
				inCue = true
			}
		} else if line == "" {
			// Empty line marks end of subtitle
			if inCue && len(currentText) > 0 {
				subtitles = append(subtitles, SRTSubtitle{
					Index:     index,
					StartTime: currentStart,
					EndTime:   currentEnd,
					Text:      strings.Join(currentText, "\n"),
				})
				index++
				currentText = []string{}
			}
			inCue = false
		} else if inCue && line != "" {
			// Remove WebVTT tags like <v Name> or <c>
			cleanLine := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(line, "")
			if cleanLine != "" {
				currentText = append(currentText, cleanLine)
			}
		}
	}

	// Add last subtitle if exists
	if inCue && len(currentText) > 0 {
		subtitles = append(subtitles, SRTSubtitle{
			Index:     index,
			StartTime: currentStart,
			EndTime:   currentEnd,
			Text:      strings.Join(currentText, "\n"),
		})
	}

	if len(subtitles) == 0 {
		return "", errors.New("no subtitle entries found in WebVTT")
	}

	return convertToSRT(subtitles), nil
}

// convertToSRT converts subtitle entries to SRT format string
func convertToSRT(subtitles []SRTSubtitle) string {
	var srtBuilder strings.Builder

	for i, sub := range subtitles {
		// Re-index in case of gaps
		srtBuilder.WriteString(fmt.Sprintf("%d\n", i+1))
		srtBuilder.WriteString(fmt.Sprintf("%s --> %s\n",
			formatSRTTime(sub.StartTime),
			formatSRTTime(sub.EndTime)))
		srtBuilder.WriteString(sub.Text)
		srtBuilder.WriteString("\n\n")
	}

	return strings.TrimSpace(srtBuilder.String())
}

// parseTimeCode parses TTML timecode formats
// Supports: HH:MM:SS.mmm, HH:MM:SS:fff (frames), SS.mmms
func parseTimeCode(timeCode string) (time.Duration, error) {
	// Remove any 't' prefix if present
	timeCode = strings.TrimPrefix(timeCode, "t")

	// Handle frames format (HH:MM:SS:FF)
	if strings.Count(timeCode, ":") == 3 {
		parts := strings.Split(timeCode, ":")
		if len(parts) != 4 {
			return 0, fmt.Errorf("invalid timecode format: %s", timeCode)
		}

		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		seconds, _ := strconv.Atoi(parts[2])
		frames, _ := strconv.Atoi(parts[3])

		// Assume 30 fps for frame conversion
		milliseconds := (frames * 1000) / 30

		duration := time.Duration(hours)*time.Hour +
			time.Duration(minutes)*time.Minute +
			time.Duration(seconds)*time.Second +
			time.Duration(milliseconds)*time.Millisecond

		return duration, nil
	}

	// Handle HH:MM:SS.mmm format
	if strings.Contains(timeCode, ":") {
		parts := strings.Split(timeCode, ":")
		if len(parts) < 2 {
			return 0, fmt.Errorf("invalid timecode format: %s", timeCode)
		}

		hours := 0
		minutes := 0
		seconds := 0.0

		if len(parts) == 3 {
			hours, _ = strconv.Atoi(parts[0])
			minutes, _ = strconv.Atoi(parts[1])
			seconds, _ = strconv.ParseFloat(parts[2], 64)
		} else if len(parts) == 2 {
			minutes, _ = strconv.Atoi(parts[0])
			seconds, _ = strconv.ParseFloat(parts[1], 64)
		}

		duration := time.Duration(hours)*time.Hour +
			time.Duration(minutes)*time.Minute +
			time.Duration(seconds*float64(time.Second))

		return duration, nil
	}

	// Handle seconds format (SS.mmms or just milliseconds)
	if strings.HasSuffix(timeCode, "s") {
		timeCode = strings.TrimSuffix(timeCode, "s")
		seconds, err := strconv.ParseFloat(timeCode, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(seconds * float64(time.Second)), nil
	}

	// Handle plain milliseconds
	ms, err := strconv.ParseFloat(timeCode, 64)
	if err != nil {
		return 0, err
	}

	return time.Duration(ms) * time.Millisecond, nil
}

// parseWebVTTTime parses WebVTT timestamp format (HH:MM:SS.mmm or MM:SS.mmm)
func parseWebVTTTime(timestamp string) (time.Duration, error) {
	// Remove any position/alignment info after space
	if idx := strings.Index(timestamp, " "); idx != -1 {
		timestamp = timestamp[:idx]
	}

	parts := strings.Split(timestamp, ":")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid WebVTT timestamp: %s", timestamp)
	}

	var hours, minutes int
	var seconds float64
	var err error

	if len(parts) == 3 {
		// HH:MM:SS.mmm format
		hours, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		minutes, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
		seconds, err = strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return 0, err
		}
	} else {
		// MM:SS.mmm format
		minutes, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		seconds, err = strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, err
		}
	}

	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds*float64(time.Second))

	return duration, nil
}

// formatSRTTime formats duration to SRT timestamp format (HH:MM:SS,mmm)
func formatSRTTime(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	milliseconds := int(d.Milliseconds()) % 1000

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, milliseconds)
}

// SaveToFile writes subtitle content to a file
func SaveToFile(content, filePath string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

// ExtractClosedCaptionsFromMP4 extracts EIA-608/CEA-608 closed captions from MP4 file using FFmpeg
func ExtractClosedCaptionsFromMP4(videoPath, outputPath, ffmpegPath string) error {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	// Check if video file exists
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return fmt.Errorf("video file not found: %s", videoPath)
	}

	// Extract closed captions to SRT format
	// FFmpeg can extract EIA-608 captions directly to SRT
	cmd := exec.Command(ffmpegPath,
		"-i", videoPath,
		"-map", "0:s?", // Select subtitle streams (including closed captions)
		"-c:s", "srt", // Convert to SRT
		"-f", "srt", // Force SRT format
		"-y", // Overwrite output
		outputPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Try alternative method for closed captions
		return extractClosedCaptionsAlternative(videoPath, outputPath, ffmpegPath)
	}

	// Check if output file was created and has content
	info, err := os.Stat(outputPath)
	if err != nil || info.Size() == 0 {
		return extractClosedCaptionsAlternative(videoPath, outputPath, ffmpegPath)
	}

	return nil
}

// extractClosedCaptionsAlternative uses alternative FFmpeg method to extract CC
func extractClosedCaptionsAlternative(videoPath, outputPath, ffmpegPath string) error {
	// Try extracting using subtitle filter
	cmd := exec.Command(ffmpegPath,
		"-f", "lavfi",
		"-i", fmt.Sprintf("movie=%s[out+subcc]", videoPath),
		"-map", "0:s",
		"-y",
		outputPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Try additional FFmpeg methods without external dependencies
		return extractWithFFmpegOnly(videoPath, outputPath, ffmpegPath)
	}

	// Check if output file was created and has content
	info, err := os.Stat(outputPath)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("no closed captions found in video or failed to extract")
	}

	return nil
}

// extractWithFFmpegOnly uses FFmpeg-only methods without external dependencies like CCExtractor
func extractWithFFmpegOnly(videoPath, outputPath, ffmpegPath string) error {
	// Method 1: Try extracting with text subtitles codec
	cmd := exec.Command(ffmpegPath,
		"-i", videoPath,
		"-map", "0:s:0",
		"-c:s", "text",
		"-f", "srt",
		"-y",
		outputPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		if info, statErr := os.Stat(outputPath); statErr == nil && info.Size() > 0 {
			return nil
		}
	}

	// Method 2: Try extracting with data stream (for EIA-608/CEA-608)
	cmd = exec.Command(ffmpegPath,
		"-i", videoPath,
		"-c:s", "mov_text",
		"-f", "srt",
		"-y",
		outputPath)

	stderr.Reset()
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err == nil {
		if info, statErr := os.Stat(outputPath); statErr == nil && info.Size() > 0 {
			return nil
		}
	}

	// Method 3: Try extracting all available subtitle streams
	cmd = exec.Command(ffmpegPath,
		"-txt_format", "text",
		"-i", videoPath,
		"-map", "0:s?",
		"-y",
		outputPath)

	stderr.Reset()
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("closed caption extraction failed: no subtitle streams found in video")
	}

	// Check if output file was created and has content
	info, err := os.Stat(outputPath)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("no closed captions found in video")
	}

	return nil
}

// HasClosedCaptions checks if a video file contains closed captions
func HasClosedCaptions(videoPath, ffmpegPath string) (bool, error) {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	// Use ffprobe to check for subtitle/closed caption streams
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "s",
		videoPath)

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Parse JSON output
	var result struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
			CodecType string `json:"codec_type"`
			Tags      struct {
				Language string `json:"language"`
			} `json:"tags"`
		} `json:"streams"`
	}

	err = json.Unmarshal(output, &result)
	if err != nil {
		return false, err
	}

	return len(result.Streams) > 0, nil
}

// ExtractAllSubtitlesFromMP4 extracts all subtitle/CC tracks from MP4 file
func ExtractAllSubtitlesFromMP4(videoPath, outputDir, ffmpegPath string) ([]string, error) {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}

	// Get all subtitle stream info
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "s",
		videoPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to probe video: %v", err)
	}

	var result struct {
		Streams []struct {
			Index     int    `json:"index"`
			CodecName string `json:"codec_name"`
			CodecType string `json:"codec_type"`
			Tags      struct {
				Language string `json:"language"`
			} `json:"tags"`
		} `json:"streams"`
	}

	err = json.Unmarshal(output, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %v", err)
	}

	if len(result.Streams) == 0 {
		return nil, fmt.Errorf("no subtitle streams found")
	}

	var outputFiles []string

	// Extract each subtitle stream
	for i, stream := range result.Streams {
		lang := stream.Tags.Language
		if lang == "" {
			lang = "unknown"
		}

		outputFile := fmt.Sprintf("%s/%s_%d.srt", outputDir, lang, i)

		cmd := exec.Command(ffmpegPath,
			"-i", videoPath,
			"-map", fmt.Sprintf("0:%d", stream.Index),
			"-c:s", "srt",
			"-y",
			outputFile)

		if err := cmd.Run(); err != nil {
			continue
		}

		// Verify file was created
		if info, err := os.Stat(outputFile); err == nil && info.Size() > 0 {
			outputFiles = append(outputFiles, outputFile)
		}
	}

	if len(outputFiles) == 0 {
		return nil, fmt.Errorf("failed to extract any subtitle tracks")
	}

	return outputFiles, nil
}

// CleanSRTFile removes duplicate entries and fixes formatting issues
func CleanSRTFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Remove WebVTT/ASS formatting tags
	content = removeFormattingTags(content)

	lines := strings.Split(string(content), "\n")
	var cleaned []string
	var currentEntry []string
	seenText := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" && len(currentEntry) > 0 {
			// End of subtitle entry
			if len(currentEntry) >= 3 {
				text := strings.Join(currentEntry[2:], "\n")
				if !seenText[text] {
					seenText[text] = true
					cleaned = append(cleaned, strings.Join(currentEntry, "\n"))
					cleaned = append(cleaned, "")
				}
			}
			currentEntry = nil
		} else if line != "" {
			currentEntry = append(currentEntry, line)
		}
	}

	// Add last entry if exists
	if len(currentEntry) >= 3 {
		text := strings.Join(currentEntry[2:], "\n")
		if !seenText[text] {
			cleaned = append(cleaned, strings.Join(currentEntry, "\n"))
		}
	}

	return os.WriteFile(filePath, []byte(strings.Join(cleaned, "\n")), 0644)
}

// removeFormattingTags removes WebVTT and ASS formatting tags from subtitle content
func removeFormattingTags(content []byte) []byte {
	text := string(content)

	// Remove WebVTT/ASS positioning tags: {\an1} to {\an9}
	text = regexp.MustCompile(`\{\\an\d\}`).ReplaceAllString(text, "")

	// Remove other common ASS/WebVTT tags
	text = regexp.MustCompile(`\{\\[^}]+\}`).ReplaceAllString(text, "")

	// Remove HTML-like tags: <b>, <i>, <u>, <font>, etc.
	text = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(text, "")

	// Remove WebVTT voice tags: <v Name>
	text = regexp.MustCompile(`<v\s+[^>]+>`).ReplaceAllString(text, "")

	// Remove WebVTT class tags: <c.className>
	text = regexp.MustCompile(`<c\.[^>]+>`).ReplaceAllString(text, "")

	// Remove positioning cues like 'position:' and 'align:'
	text = regexp.MustCompile(`(?m)^.*(?:position|align|line|size):[^\n]*\n`).ReplaceAllString(text, "")

	// Remove escape sequences like \h, \n, \t (literal backslash followed by letter)
	text = strings.ReplaceAll(text, "\\h", "")
	text = regexp.MustCompile(`\\[a-z]`).ReplaceAllString(text, "")

	// Remove patterns like /h/h/h that might appear
	text = regexp.MustCompile(`(/[a-z])+`).ReplaceAllString(text, "")

	return []byte(text)
}

// ConvertSCCToSRT converts SCC (Scenarist Closed Caption) format to SRT
func ConvertSCCToSRT(sccContent string) (string, error) {
	// SCC is a hex-based format, FFmpeg handles this better
	// This is a placeholder for direct conversion if needed
	return "", errors.New("SCC to SRT conversion requires FFmpeg or CCExtractor")
}
