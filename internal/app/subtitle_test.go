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
