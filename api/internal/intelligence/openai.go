package intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIConfig configures an OpenAI-compatible chat completions endpoint.
type OpenAIConfig struct {
	APIKey  string
	Model   string
	BaseURL string // default https://api.openai.com/v1
}

// OpenAI implements Summarizer via chat completions (OpenAI, Ollama, etc.).
type OpenAI struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewOpenAI(cfg OpenAIConfig) *OpenAI {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &OpenAI{
		apiKey:  cfg.APIKey,
		model:   model,
		baseURL: base,
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (o *OpenAI) ModelVersion() string {
	return "openai:" + o.model
}

func (o *OpenAI) SummarizeThread(ctx context.Context, in ThreadSummaryInput) (string, error) {
	prompt := formatThreadForPrompt(in)
	if prompt == "" {
		return "", fmt.Errorf("empty thread content")
	}

	body := map[string]any{
		"model": o.model,
		"messages": []map[string]string{
			{
				"role": "system",
				"content": "You summarize community forum threads for newcomers and search engines. " +
					"Write 2–3 concise sentences. Stick to facts stated in the posts. No speculation or advice not in the thread.",
			},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"max_tokens":  280,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	res, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("llm api %s: %s", res.Status, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llm api: empty choices")
	}
	out := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if out == "" {
		return "", fmt.Errorf("llm api: empty summary")
	}
	return out, nil
}

func formatThreadForPrompt(in ThreadSummaryInput) string {
	var b strings.Builder
	title := strings.TrimSpace(in.Title)
	if title != "" {
		b.WriteString("Thread title: ")
		b.WriteString(title)
		b.WriteString("\n\n")
	}
	for _, p := range in.Posts {
		body := strings.TrimSpace(p.BodyPlain)
		if body == "" {
			continue
		}
		if p.IsOP {
			b.WriteString("Original post (")
		} else {
			b.WriteString("Reply (")
		}
		b.WriteString(p.Author)
		b.WriteString("):\n")
		b.WriteString(body)
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}