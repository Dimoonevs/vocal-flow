package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	libVideo "github.com/Dimoonevs/video-service/app/pkg/lib"
	"github.com/Dimoonevs/vocal-flow/app/internal/models"
	"github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

const openAIKey = "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"

const openAITranslateURL = "https://api.openai.com/v1/chat/completions"

func TranscribeVideo(filePath string, userSetting *models.UserSettings) (*models.WhisperResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	_ = writer.WriteField("model", userSetting.WhisperModel)
	_ = writer.WriteField("response_format", "verbose_json")
	_ = writer.WriteField("timestamp_granularities[]", "segment")

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+userSetting.AIToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var response *models.WhisperResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return response, nil
}

func TranslateText(text, targetLang string, userSetting *models.UserSettings) (string, error) {
	requestBody, err := json.Marshal(models.OpenAIRequest{
		Model: userSetting.GPTModel,
		Messages: []models.Message{
			{Role: "system", Content: fmt.Sprintf("Translate the following text to %s and remove all punctuation marks. Respond with only the translated text and nothing else.", targetLang)},
			{Role: "user", Content: text},
		},
		MaxTokens: 500,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", openAITranslateURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userSetting.AIToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: %s", body)
	}

	var result models.OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("no translation found in response")
	}

	return result.Choices[0].Message.Content, nil
}

func GetSummary(text, lang string, userSetting *models.UserSettings) (string, error) {
	requestBody, err := json.Marshal(models.OpenAIRequest{
		Model: userSetting.GPTModel,
		Messages: []models.Message{
			{Role: "system", Content: fmt.Sprintf("Briefly summarize the following text. The output language should be: %s", lang)},
			{Role: "user", Content: text},
		},
		MaxTokens: 300,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", openAITranslateURL, bytes.NewBuffer(requestBody))
	if err != nil {
		logrus.Errorf("failed to create request: %s", err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userSetting.AIToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("failed to send request: %s", err)
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logrus.Errorf("OpenAI API error: %s", body)
		return "", fmt.Errorf("OpenAI API error: %s", body)
	}

	var result models.OpenAIResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logrus.Errorf("failed to decode response: %s", err)
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		logrus.Error("no summary found in response")
		return "", fmt.Errorf("no summary found in response")
	}

	return result.Choices[0].Message.Content, nil
}

func TransformDataAI(data *models.DataAI) {
	if data.SubtitlesURL.Valid {
		// Преобразуем JSON-массив строк в структуру
		var subs []struct {
			URI  string `json:"uri"`
			Lang string `json:"lang"`
		}
		if err := json.Unmarshal([]byte(data.SubtitlesURL.String), &subs); err == nil {
			for i := range subs {
				subs[i].URI = libVideo.GetVideoPublicLink(subs[i].URI)
			}
			// Сохраняем обратно преобразованный JSON
			if updated, err := json.Marshal(subs); err == nil {
				data.SubtitlesURL.String = string(updated)
			}
		}
	}

	if data.SubtitlesVideoURL.Valid {
		data.SubtitlesVideoURL.String = libVideo.GetVideoPublicLink(data.SubtitlesVideoURL.String)
	}

	if data.TranslateVideoURL.Valid {
		data.TranslateVideoURL.String = libVideo.GetVideoPublicLink(data.TranslateVideoURL.String)
	}
}
