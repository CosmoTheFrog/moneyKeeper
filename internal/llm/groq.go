// Package llm содержит клиенты языковых моделей. Содержимое запросов в логи и в тексты ошибок не попадает.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	groqURL     = "https://api.groq.com"
	groqModel   = "llama-3.3-70b-versatile"
	groqTimeout = 60 * time.Second
	maxReply    = 1 << 20
)

// ErrUnavailable: модель не ответила или ответила не 200. Подробности без тела запроса.
var ErrUnavailable = errors.New("llm unavailable")

type Groq struct {
	base   string
	key    string
	client *http.Client
}

func NewGroq(key string) *Groq {
	return &Groq{base: groqURL, key: key, client: &http.Client{Timeout: groqTimeout}}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	Temperature    float64       `json:"temperature"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// CompleteJSON отправляет системную и пользовательскую часть и возвращает ответ модели, который гарантированно JSON.
func (g *Groq) CompleteJSON(ctx context.Context, system, user string) (string, error) {
	req := chatRequest{
		Model:    groqModel,
		Messages: []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
	}
	req.ResponseFormat.Type = "json_object"
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.base+"/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.key)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		// Ошибка клиента может содержать адрес, но не тело запроса.
		return "", fmt.Errorf("%w: request failed", ErrUnavailable)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}

	var out chatResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxReply)).Decode(&out); err != nil || len(out.Choices) == 0 {
		return "", fmt.Errorf("%w: bad reply", ErrUnavailable)
	}
	return out.Choices[0].Message.Content, nil
}
