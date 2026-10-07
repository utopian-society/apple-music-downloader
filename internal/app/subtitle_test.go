package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanSRTText(t *testing.T) {
	input := `1
00:00:00,000 --> 00:00:02,266
<font face="Monospace">{\an7}[WATER RUNNING]</font>

2
00:00:19,976 --> 00:00:24,849
<font face="Monospace">{\an7}\h\h\h\h♪ HAND TO GOD
I PROMISED I TRIED ♪</font>
`
	cleaned := CleanSRTText(input)

	if strings.Contains(cleaned, "<font") || strings.Contains(cleaned, "</font>") {
		t.Errorf("CleanSRTText should remove font tags: %s", cleaned)
	}
	if strings.Contains(cleaned, `{\an7}`) {
		t.Errorf("CleanSRTText should remove ASS positioning tags: %s", cleaned)
	}
	if strings.Contains(cleaned, `\h`) {
		t.Errorf("CleanSRTText should convert \\h to space: %s", cleaned)
	}
	if !strings.Contains(cleaned, "[WATER RUNNING]") {
		t.Errorf("CleanSRTText should keep text: %s", cleaned)
	}
	if !strings.Contains(cleaned, "♪ HAND TO GOD") {
		t.Errorf("CleanSRTText should keep lyrics: %s", cleaned)
	}
}

func TestValidateSRT(t *testing.T) {
	tmpDir := t.TempDir()
	validSRT := filepath.Join(tmpDir, "valid.srt")
	err := os.WriteFile(validSRT, []byte("1\n00:00:01,000 --> 00:00:04,000\nHello World\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	if err := ValidateSRT(validSRT); err != nil {
		t.Errorf("ValidateSRT should pass for valid file: %v", err)
	}

	nonExistent := filepath.Join(tmpDir, "does_not_exist.srt")
	if err := ValidateSRT(nonExistent); err == nil {
		t.Errorf("ValidateSRT should fail for non-existent file")
	}
}

func TestParseSRTSubtitles(t *testing.T) {
	tmpDir := t.TempDir()
	srtFile := filepath.Join(tmpDir, "test.srt")
	content := "1\n00:00:01,000 --> 00:00:04,000\nHello World\n"
	if err := os.WriteFile(srtFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseSRTSubtitles(srtFile)
	if err != nil {
		t.Fatalf("ParseSRTSubtitles error: %v", err)
	}
	if !strings.Contains(parsed, "Hello World") {
		t.Errorf("ParseSRTSubtitles content missing: %s", parsed)
	}
}

func TestBuildFFmpegEmbedArgsSingle(t *testing.T) {
	subs := []SubtitleTrack{
		{
			Path:     "/path/to/sub.vtt",
			Language: "en-US",
			Title:    "English",
		},
	}
	args := BuildFFmpegEmbedArgs("/path/to/input.mp4", subs, "/path/to/output.mp4")

	expected := []string{
		"-y", "-i", "/path/to/input.mp4",
		"-i", "/path/to/sub.vtt",
		"-map", "0",
		"-map", "1",
		"-c", "copy", "-c:s", "mov_text",
		"-metadata:s:s:0", "language=eng",
		"-metadata:s:s:0", "title=English",
		"/path/to/output.mp4",
	}

	if len(args) != len(expected) {
		t.Fatalf("unexpected args length: got %d, want %d\nGot: %v\nWant: %v", len(args), len(expected), args, expected)
	}
	for i := range args {
		if args[i] != expected[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, args[i], expected[i])
		}
	}
}

func TestBuildFFmpegEmbedArgsMultiple(t *testing.T) {
	subs := []SubtitleTrack{
		{
			Path:     "/tmp/sub_en.vtt",
			Language: "en",
			Title:    "English",
		},
		{
			Path:     "/tmp/sub_ja.vtt",
			Language: "ja",
			Title:    "Japanese",
		},
	}
	args := BuildFFmpegEmbedArgs("/tmp/in.mp4", subs, "/tmp/out.mp4")

	expected := []string{
		"-y", "-i", "/tmp/in.mp4",
		"-i", "/tmp/sub_en.vtt",
		"-i", "/tmp/sub_ja.vtt",
		"-map", "0",
		"-map", "1",
		"-map", "2",
		"-c", "copy", "-c:s", "mov_text",
		"-metadata:s:s:0", "language=eng",
		"-metadata:s:s:0", "title=English",
		"-metadata:s:s:1", "language=jpn",
		"-metadata:s:s:1", "title=Japanese",
		"/tmp/out.mp4",
	}

	if len(args) != len(expected) {
		t.Fatalf("unexpected args length: got %d, want %d\nGot: %v\nWant: %v", len(args), len(expected), args, expected)
	}
	for i := range args {
		if args[i] != expected[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, args[i], expected[i])
		}
	}
}

func TestNormalizeISO6392(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"en", "eng"},
		{"en-US", "eng"},
		{"EN_gb", "eng"},
		{"ja", "jpn"},
		{"ja-JP", "jpn"},
		{"es", "spa"},
		{"fr-FR", "fra"},
		{"zh-Hans", "zho"},
		{"ko", "kor"},
		{"eng", "eng"},
		{"jpn", "jpn"},
		{"", "und"},
	}

	for _, tc := range tests {
		got := NormalizeISO6392(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeISO6392(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFilterSubtitles(t *testing.T) {
	tracks := []SubtitleTrack{
		{Language: "en-US", Title: "English"},
		{Language: "ja", Title: "Japanese"},
		{Language: "es", Title: "Spanish"},
		{Language: "fr", Title: "French"},
	}

	// 1. Empty filter returns all
	all := FilterSubtitles(tracks, nil)
	if len(all) != 4 {
		t.Fatalf("expected 4 tracks, got %d", len(all))
	}

	// 2. Filter by ISO 639-1 code
	enOnly := FilterSubtitles(tracks, []string{"en"})
	if len(enOnly) != 1 || enOnly[0].Language != "en-US" {
		t.Fatalf("expected en-US track, got: %v", enOnly)
	}

	// 3. Filter by multiple languages (ISO 639-2 and name)
	multi := FilterSubtitles(tracks, []string{"jpn", "spanish"})
	if len(multi) != 2 {
		t.Fatalf("expected 2 tracks, got %d: %v", len(multi), multi)
	}

	// 4. Non-matching filter returns empty
	none := FilterSubtitles(tracks, []string{"de"})
	if len(none) != 0 {
		t.Fatalf("expected 0 tracks, got %d: %v", len(none), none)
	}
}

func TestSrtToWebVTT(t *testing.T) {
	srt := "1\n00:00:01,234 --> 00:00:05,678\nHello world\n"
	vtt := srtToWebVTT(srt)
	if !strings.HasPrefix(vtt, "WEBVTT") {
		t.Fatalf("expected WEBVTT header, got: %s", vtt)
	}
	if !strings.Contains(vtt, "00:00:01.234 --> 00:00:05.678") {
		t.Fatalf("expected converted timestamps with dots, got: %s", vtt)
	}
	if !strings.Contains(vtt, "Hello world") {
		t.Fatalf("expected payload preserved, got: %s", vtt)
	}
}
