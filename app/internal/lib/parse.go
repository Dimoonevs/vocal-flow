package lib

import (
	"fmt"
	"github.com/Dimoonevs/vocal-flow/app/internal/models"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func formatSRTTime(seconds float64) string {
	t := time.Duration(seconds * float64(time.Second))
	hours := int(t.Hours())
	minutes := int(t.Minutes()) % 60
	secondsInt := int(t.Seconds()) % 60
	milliseconds := t.Milliseconds() % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secondsInt, milliseconds)
}

func SaveSRT(lang string, segments []models.TranslatedSegment, outputDir string) (string, error) {
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	filePath := filepath.Join(outputDir, fmt.Sprintf("subtitles_%s.srt", lang))
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create SRT file: %w", err)
	}
	defer file.Close()

	for i, segment := range segments {
		startTime := formatSRTTime(segment.Start)
		endTime := formatSRTTime(segment.End)
		_, err := fmt.Fprintf(file, "%d\n%s --> %s\n%s\n\n", i+1, startTime, endTime, segment.Text)
		if err != nil {
			return "", fmt.Errorf("failed to write to SRT file: %w", err)
		}
	}

	logrus.Infof("SRT saved: %s", filePath)
	return filePath, nil
}

func ReadSRTFile(filepath string) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		logrus.Errorf("Failed to read SRT file: %s", err)
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	text := string(content)

	re := regexp.MustCompile(`\d+\n\d{2}:\d{2}:\d{2},\d{3} --> \d{2}:\d{2}:\d{2},\d{3}\n`)
	cleanText := re.ReplaceAllString(text, "")

	lines := strings.Split(cleanText, "\n")
	var filtered []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			filtered = append(filtered, strings.TrimSpace(line))
		}
	}
	return strings.Join(filtered, " "), nil
}
