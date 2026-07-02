package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr        string
	DatabaseURL string
	MediaDir    string
	EditWindow  time.Duration
	LLM         LLMConfig
}

// LLMConfig is optional. No API key → heuristic summaries only.
type LLMConfig struct {
	Provider string // openai (OpenAI-compatible chat completions)
	APIKey   string
	Model    string
	BaseURL  string // override for Ollama, Azure, etc.
}

func (c LLMConfig) Enabled() bool {
	return strings.TrimSpace(c.APIKey) != "" && strings.TrimSpace(c.Provider) != ""
}

const defaultEditWindow = 48 * time.Hour

func Load() Config {
	addr := os.Getenv("XUROI_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://xuroi:xuroi_dev@localhost:5433/xuroi?sslmode=disable"
	}

	mediaDir := os.Getenv("MEDIA_DIR")
	if mediaDir == "" {
		mediaDir = "../infra/uploads"
	}

	return Config{
		Addr:        addr,
		DatabaseURL: dbURL,
		MediaDir:    mediaDir,
		EditWindow:  editWindowFromEnv(),
		LLM:         llmFromEnv(),
	}
}

func llmFromEnv() LLMConfig {
	return LLMConfig{
		Provider: os.Getenv("XUROI_LLM_PROVIDER"),
		APIKey:   os.Getenv("XUROI_LLM_API_KEY"),
		Model:    os.Getenv("XUROI_LLM_MODEL"),
		BaseURL:  os.Getenv("XUROI_LLM_BASE_URL"),
	}
}

func editWindowFromEnv() time.Duration {
	if v := os.Getenv("EDIT_WINDOW_HOURS"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			return time.Duration(h) * time.Hour
		}
	}
	return defaultEditWindow
}