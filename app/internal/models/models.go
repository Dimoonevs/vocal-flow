package models

type Video struct {
	URI string `json:"uri"`
}

type TranscriptionRequest struct {
	ID    int      `json:"id"`
	Langs []string `json:"langs,omitempty"`
}
type WhisperResponse struct {
	Task     string     `json:"task"`
	Language string     `json:"language"`
	Duration float64    `json:"duration"`
	Text     string     `json:"text"`
	Segments []Segments `json:"segments"`
}

type Segments struct {
	Id               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

type TranslatedSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type RequestDeepl struct {
	TargetLanguage string   `json:"target_lang"`
	Text           []string `json:"text"`
}

type OpenAIRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type SubtitlesData struct {
	URI  string `json:"uri"`
	Lang string `json:"lang"`
}
