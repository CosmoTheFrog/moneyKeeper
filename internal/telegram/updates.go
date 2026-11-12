package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// pollTimeout это время, которое Telegram держит запрос getUpdates открытым без новых сообщений.
const pollTimeout = 25

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type Message struct {
	ID   int64  `json:"message_id"`
	From *User  `json:"from"`
	Chat Chat   `json:"chat"`
	Text string `json:"text"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

// Update это одно событие: сообщение или нажатие кнопки под сообщением бота.
type Update struct {
	ID       int64          `json:"update_id"`
	Message  *Message       `json:"message"`
	Callback *CallbackQuery `json:"callback_query"`
}

// Button это кнопка под сообщением: Data вернётся в CallbackQuery при нажатии (до 64 байт).
type Button struct {
	Text string `json:"text"`
	Data string `json:"callback_data"`
}

// call отправляет запрос методу Bot API и разбирает result. Ошибка содержит только статус, без адреса с токеном.
func (c *Client) call(ctx context.Context, hc *http.Client, method string, payload, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: encode: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/bot"+c.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: request: %w", errNoURL(err))
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: %s: %w", method, errNoURL(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: %s: status %d", method, resp.StatusCode)
	}
	if result == nil {
		return nil
	}
	var out struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("telegram: %s: decode reply", method)
	}
	if err := json.Unmarshal(out.Result, result); err != nil {
		return fmt.Errorf("telegram: %s: decode result", method)
	}
	return nil
}

// Updates ждёт новые события после offset (long polling). Для следующего вызова offset это id последнего события плюс один.
func (c *Client) Updates(ctx context.Context, offset int64) ([]Update, error) {
	hc := &http.Client{Timeout: (pollTimeout + 15) * time.Second}
	var out []Update
	err := c.call(ctx, hc, "getUpdates", map[string]any{
		"offset": offset, "timeout": pollTimeout, "allowed_updates": []string{"message", "callback_query"},
	}, &out)
	return out, err
}

// SendButtons отправляет текст с рядом кнопок под ним и возвращает номер сообщения.
func (c *Client) SendButtons(ctx context.Context, chatID int64, text string, rows [][]Button) (int64, error) {
	var msg Message
	err := c.call(ctx, c.http, "sendMessage", map[string]any{
		"chat_id": chatID, "text": text, "reply_markup": map[string]any{"inline_keyboard": rows},
	}, &msg)
	return msg.ID, err
}

// EditText заменяет текст сообщения и его кнопки (пустой rows убирает кнопки).
func (c *Client) EditText(ctx context.Context, chatID, messageID int64, text string, rows [][]Button) error {
	if rows == nil {
		rows = [][]Button{}
	}
	return c.call(ctx, c.http, "editMessageText", map[string]any{
		"chat_id": chatID, "message_id": messageID, "text": text, "reply_markup": map[string]any{"inline_keyboard": rows},
	}, nil)
}

// AnswerCallback снимает «часики» с нажатой кнопки.
func (c *Client) AnswerCallback(ctx context.Context, callbackID string) error {
	return c.call(ctx, c.http, "answerCallbackQuery", map[string]any{"callback_query_id": callbackID}, nil)
}
