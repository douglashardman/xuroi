package intelligence

import (
	"strings"

	"github.com/xuroi/xuroi/api/internal/config"
)

// NewSummarizerFromConfig returns an LLM summarizer when env is configured, else nil.
func NewSummarizerFromConfig(cfg config.LLMConfig) Summarizer {
	if !cfg.Enabled() {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "openai", "ollama":
		baseURL := cfg.BaseURL
		if baseURL == "" && strings.EqualFold(cfg.Provider, "ollama") {
			baseURL = "http://localhost:11434/v1"
		}
		return NewOpenAI(OpenAIConfig{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: baseURL,
		})
	default:
		return nil
	}
}