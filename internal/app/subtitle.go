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
	"sync"
	"time"
)

var (
	ffmpegCheckOnce sync.Once
	ffmpegFound     bool
)

// isFFmpegAvailable checks once if ffmpeg is in PATH, logs a warning if missing,
// and returns true if available.
func isFFmpegAvailable() bool {
	ffmpegCheckOnce.Do(func() {
		_, err := exec.LookPath("ffmpeg")
		ffmpegFound = (err == nil)
		if !ffmpegFound {
			fmt.Println("Warning: ffmpeg not found in PATH; subtitle embedding and conversion will be skipped.")
		}
	})
	return ffmpegFound
}

// SubtitleTrack represents a subtitle file to be embedded into an MP4 container.
type SubtitleTrack struct {
	Path     string
	Language string
	Title    string
}

// SubtitleRendition represents a subtitle rendition declared in an HLS master playlist.
type SubtitleRendition struct {
	Language string
	Name     string
	URL      string
}

// ExtractedSubtitle represents a subtitle track extracted or downloaded for an MV.
type ExtractedSubtitle struct {
	Lang string
	Path string
}

// NormalizeISO6392 normalizes BCP-47 or 2-letter ISO 639-1 language tags into 3-letter ISO 639-2 codes.
func NormalizeISO6392(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if len(lang) == 0 {
		return "und"
	}
	base := lang
	if idx := strings.IndexAny(base, "-_"); idx != -1 {
		base = base[:idx]
	}

	switch base {
	case "en":
		return "eng"
	case "es":
		return "spa"
	case "fr":
		return "fra"
	case "de":
		return "deu"
	case "it":
		return "ita"
	case "pt":
		return "por"
	case "ja":
		return "jpn"
	case "zh":
		return "zho"
	case "ko":
		return "kor"
	case "ru":
		return "rus"
	case "ar":
		return "ara"
	case "hi":
		return "hin"
	case "nl":
		return "nld"
	case "pl":
		return "pol"
	case "sv":
		return "swe"
	case "tr":
		return "tur"
	case "vi":
		return "vie"
	case "th":
		return "tha"
	case "id":
		return "ind"
	case "el":
		return "ell"
	case "he":
		return "heb"
	case "da":
		return "dan"
	case "fi":
		return "fin"
	case "no":
		return "nor"
	case "english":
		return "eng"
	case "spanish":
		return "spa"
	case "french":
		return "fra"
	case "german":
		return "deu"
	case "italian":
		return "ita"
	case "portuguese":
		return "por"
	case "japanese":
		return "jpn"
	case "chinese":
		return "zho"
	case "korean":
		return "kor"
	case "russian":
		return "rus"
	}

	if len(lang) == 3 {
		return lang
	}
	if len(base) == 3 {
		return base
	}
	return lang
}

func matchesLanguage(lang, title, filter string) bool {
	filter = strings.TrimSpace(strings.ToLower(filter))
	if filter == "" {
		return true
	}
	l := strings.TrimSpace(strings.ToLower(lang))
	t := strings.TrimSpace(strings.ToLower(title))
	normLang := NormalizeISO6392(l)
	normFilter := NormalizeISO6392(filter)

	if l == filter || t == filter || normLang == normFilter {
		return true
	}
	if strings.HasPrefix(l, filter+"-") || strings.HasPrefix(l, filter+"_") {
		return true
	}
	for _, w := range strings.Fields(t) {
		cleanW := strings.Trim(w, ",.-_()[]")
		if cleanW == filter || NormalizeISO6392(cleanW) == normFilter {
			return true
		}
	}
	return false
}

func filterItem(lang, title string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if matchesLanguage(lang, title, a) {
			return true
		}
	}
	return false
}

// FilterSubtitles filters SubtitleTrack slice by allowed language codes/names.
func FilterSubtitles(subs []SubtitleTrack, allowedLangs []string) []SubtitleTrack {
	if len(allowedLangs) == 0 {
		return subs
	}
	var res []SubtitleTrack
	for _, s := range subs {
		if filterItem(s.Language, s.Title, allowedLangs) {
			res = append(res, s)
		}
	}
	return res
}

// FilterSubtitleRenditions filters SubtitleRendition slice by allowed language codes/names.
func FilterSubtitleRenditions(subs []SubtitleRendition, allowedLangs []string) []SubtitleRendition {
	if len(allowedLangs) == 0 {
		return subs
	}
	var res []SubtitleRendition
	for _, s := range subs {
		if filterItem(s.Language, s.Name, allowedLangs) {
			res = append(res, s)
		}
	}
	return res
}

// FilterExtractedSubtitles filters ExtractedSubtitle slice by allowed language codes/names.
func FilterExtractedSubtitles(subs []ExtractedSubtitle, allowedLangs []string) []ExtractedSubtitle {
	if len(allowedLangs) == 0 {
		return subs
	}
	var res []ExtractedSubtitle
	for _, s := range subs {
		if filterItem(s.Lang, s.Lang, allowedLangs) {
			res = append(res, s)
		}
	}
	return res
}

// BuildFFmpegEmbedArgs constructs ffmpeg arguments to mux subtitle tracks into MP4 as mov_text.
func BuildFFmpegEmbedArgs(inputVideoPath string, subs []SubtitleTrack, outputVideoPath string) []string {
	args := []string{"-y", "-i", inputVideoPath}
	for _, sub := range subs {
		args = append(args, "-i", sub.Path)
	}
	args = append(args, "-map", "0")
	for i := range subs {
		args = append(args, "-map", fmt.Sprintf("%d", i+1))
	}
	args = append(args, "-c", "copy", "-c:s", "mov_text")
	for i, sub := range subs {
		lang := NormalizeISO6392(sub.Language)
		title := sub.Title
		if title == "" {
			title = sub.Language
		}
		args = append(args,
			fmt.Sprintf("-metadata:s:s:%d", i), fmt.Sprintf("language=%s", lang),
			fmt.Sprintf("-metadata:s:s:%d", i), fmt.Sprintf("title=%s", title),
		)
	}
	args = append(args, outputVideoPath)
	return args
}

// srtToWebVTT converts an SRT subtitle string into WebVTT format.
func srtToWebVTT(srt string) string {
	srt = strings.ReplaceAll(srt, "\r\n", "\n")
	srt = strings.ReplaceAll(srt, "\r", "\n")
	tsRe := regexp.MustCompile(`(\d{2}:\d{2}:\d{2}),(\d{3})\s+-->\s+(\d{2}:\d{2}:\d{2}),(\d{3})`)
	vttTimestamps := tsRe.ReplaceAllString(srt, "$1.$2 --> $3.$4")
	return "WEBVTT\n\n" + strings.TrimSpace(vttTimestamps) + "\n"
}

// EmbedSubtitlesInMV embeds subtitle tracks into an MP4 music video using ffmpeg.
func (r *Runner) EmbedSubtitlesInMV(videoPath string, subs []SubtitleTrack) error {
	if len(subs) == 0 {
		return nil
	}
	if !isFFmpegAvailable() {
		return nil
	}
	if _, err := os.Stat(videoPath); err != nil {
		return fmt.Errorf("video file not found: %w", err)
	}

	tempDir := os.TempDir()
	if r != nil && r.TempMgr != nil {
		tempDir = r.TempMgr.RootDir()
	}

	tmpOutPath := filepath.Join(tempDir, fmt.Sprintf("%s.sub_embed_%d.mp4", filepath.Base(videoPath), time.Now().UnixNano()))
	defer func() {
		if r != nil && r.TempMgr != nil {
			_ = r.TempMgr.RemoveFile(tmpOutPath)
		} else {
			_ = os.Remove(tmpOutPath)
		}
	}()

	args := BuildFFmpegEmbedArgs(videoPath, subs, tmpOutPath)
	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg subtitle embedding failed: %w, output: %s", err, string(out))
	}

	if err := replaceFile(tmpOutPath, videoPath); err != nil {
		return fmt.Errorf("replace original file with embedded subtitle output: %w", err)
	}
	return nil
}

// EmbedSubtitlesInMV embeds SRT subtitles into an MP4 music video using ffmpeg (backward compatibility).
func EmbedSubtitlesInMV(videoPath, subtitlePath, language string) (string, error) {
	if _, err := os.Stat(videoPath); err != nil {
		return "", fmt.Errorf("video file not found: %w", err)
	}
	if _, err := os.Stat(subtitlePath); err != nil {
		return "", fmt.Errorf("subtitle file not found: %w", err)
	}
	if language == "" {
		language = "eng"
	}
	subs := []SubtitleTrack{
		{
			Path:     subtitlePath,
			Language: language,
			Title:    language,
		},
	}
	r := &Runner{}
	if err := r.EmbedSubtitlesInMV(videoPath, subs); err != nil {
		return "", err
	}
	return videoPath, nil
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

		if lineNum == 1 || strings.Contains(line, " --> ") {
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to validate SRT: %w", err)
	}

	return nil
}
