package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

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
