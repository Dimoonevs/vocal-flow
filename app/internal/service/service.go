package service

import (
	"fmt"
	"github.com/Dimoonevs/vocal-flow/app/internal/lib"
	"github.com/Dimoonevs/vocal-flow/app/internal/models"
	"github.com/Dimoonevs/vocal-flow/app/internal/repo/mysql"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"sort"
	"sync"
)

func CreateTranscription(id int, langs []string) (map[string]string, error) {
	URI, err := mysql.GetConnection().GetURIByID(id)
	if err != nil {
		return nil, err
	}
	if URI == "" {
		logrus.Errorf("video %d in db not exist", id)
		return nil, fmt.Errorf("video in db not exist")
	}

	transcription, err := lib.TranscribeVideo(URI)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	dirPath := filepath.Dir(URI)
	subtitlesMap := make(map[string]string)

	originalSegments := make([]models.TranslatedSegment, 0, len(transcription.Segments))
	for _, segment := range transcription.Segments {
		originalSegments = append(originalSegments, models.TranslatedSegment{
			Start: segment.Start,
			End:   segment.End,
			Text:  segment.Text,
		})
	}

	originalFilePath, err := lib.SaveSRT("original", originalSegments, dirPath)
	if err != nil {
		logrus.Errorf("Failed to save original SRT: %v", err)
		return nil, err
	}

	subtitlesMap["original"] = originalFilePath

	var wg sync.WaitGroup
	translations := make(map[string][]models.TranslatedSegment)
	var mu sync.Mutex

	errChan := make(chan error, 1)
	var stopOnce sync.Once

	type IndexedSegment struct {
		Index   int
		Segment models.TranslatedSegment
	}

	translationResults := make(map[string][]IndexedSegment)

	for idx, segment := range transcription.Segments {
		for _, lang := range langs {
			wg.Add(1)
			go func(idx int, lang string, segment models.Segments) {
				defer wg.Done()
				translatedText, err := lib.TranslateText(segment.Text, lang)
				if err != nil {
					logrus.Errorf("Translation to %s failed: %v", lang, err)
					stopOnce.Do(func() { errChan <- fmt.Errorf("translation to %s failed: %w", lang, err) })
					return
				}

				translatedSegment := IndexedSegment{
					Index: idx,
					Segment: models.TranslatedSegment{
						Start: segment.Start,
						End:   segment.End,
						Text:  translatedText,
					},
				}

				mu.Lock()
				translationResults[lang] = append(translationResults[lang], translatedSegment)
				mu.Unlock()
			}(idx, lang, segment)
		}
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	if err, ok := <-errChan; ok {
		return nil, err
	}

	for lang, indexedSegments := range translationResults {
		sort.SliceStable(indexedSegments, func(i, j int) bool {
			return indexedSegments[i].Index < indexedSegments[j].Index
		})

		for _, seg := range indexedSegments {
			translations[lang] = append(translations[lang], seg.Segment)
		}
	}

	for lang, segments := range translations {
		wg.Add(1)
		go func(lang string, segments []models.TranslatedSegment) {
			defer wg.Done()
			filePath, err := lib.SaveSRT(lang, segments, dirPath)
			if err != nil {
				logrus.Errorf("Failed to save SRT file for %s: %v", lang, err)
				return
			}

			mu.Lock()
			subtitlesMap[lang] = filePath
			mu.Unlock()
		}(lang, segments)
	}

	wg.Wait()

	if err := mysql.GetConnection().SaveTranscription(subtitlesMap, id); err != nil {
		logrus.Errorf("Failed to save transcription: %v", err)
		return nil, err
	}

	return subtitlesMap, nil
}
