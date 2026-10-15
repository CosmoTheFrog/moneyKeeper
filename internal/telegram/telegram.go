// Package telegram отправляет сообщения через Bot API. Токен лежит в адресе запроса, поэтому в ошибки и логи адрес не попадает.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const apiURL = "https://api.telegram.org"

type Client struct {
	token string
	base  string
	http  *http.Client
}

func New(token string) *Client {
	return &Client{token: token, base: apiURL, http: &http.Client{Timeout: 10 * time.Second}}
}

// Send отправляет текст в чат. Ошибка содержит только статус, без адреса с токеном.
func (c *Client) Send(ctx context.Context, chatID int64, text string) error {
	body, err := json.Marshal(map[string]any{"chat_id": chatID, "text": text})
	if err != nil {
		return fmt.Errorf("telegram: encode: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/bot"+c.token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: request: %w", errNoURL(err))
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send: %w", errNoURL(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: status %d", resp.StatusCode)
	}
	return nil
}

// errNoURL отбрасывает адрес из ошибки http: в нём токен бота.
func errNoURL(err error) error {
	if ue, ok := err.(interface{ Unwrap() error }); ok && ue.Unwrap() != nil {
		return ue.Unwrap()
	}
	return err
}
