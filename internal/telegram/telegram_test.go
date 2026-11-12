package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &Client{token: "SECRET-TOKEN", base: srv.URL, http: srv.Client()}
}

func TestSend(t *testing.T) {
	var path string
	var body map[string]any
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
	})
	if err := c.Send(context.Background(), 42, "привет"); err != nil {
		t.Fatal(err)
	}
	if path != "/botSECRET-TOKEN/sendMessage" || body["chat_id"] != float64(42) || body["text"] != "привет" {
		t.Errorf("path %q body %v", path, body)
	}
}

func TestSendErrorsDoNotLeakTheToken(t *testing.T) {
	bad := newClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) })
	err := bad.Send(context.Background(), 1, "x")
	if err == nil || !strings.Contains(err.Error(), "400") || strings.Contains(err.Error(), "SECRET-TOKEN") {
		t.Errorf("status error = %v", err)
	}

	// Сервер недоступен: ошибка сети из http содержит адрес, но в нашей ошибке его быть не должно.
	down := &Client{token: "SECRET-TOKEN", base: "http://127.0.0.1:1", http: http.DefaultClient}
	err = down.Send(context.Background(), 1, "x")
	if err == nil || strings.Contains(err.Error(), "SECRET-TOKEN") {
		t.Errorf("network error = %v", err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := bad.Send(cancelled, 1, "x"); err == nil || strings.Contains(err.Error(), "SECRET-TOKEN") {
		t.Errorf("cancelled error = %v", err)
	}
}

func TestUpdatesAndButtons(t *testing.T) {
	var gotMethod string
	var body map[string]any
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = strings.TrimPrefix(r.URL.Path, "/botSECRET-TOKEN/")
		b, _ := io.ReadAll(r.Body)
		body = map[string]any{}
		_ = json.Unmarshal(b, &body)
		switch gotMethod {
		case "getUpdates":
			_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":7,"message":{"message_id":1,"from":{"id":42,"first_name":"I"},"chat":{"id":42,"type":"private"},"text":"кофе 5"}}]}`))
		case "sendMessage":
			_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":55,"chat":{"id":42,"type":"private"}}}`))
		}
	})

	ups, err := c.Updates(context.Background(), 8)
	if err != nil || len(ups) != 1 || ups[0].ID != 7 || ups[0].Message.Text != "кофе 5" || ups[0].Message.From.ID != 42 || body["offset"] != float64(8) {
		t.Fatalf("updates = %+v, %v, body %v", ups, err, body)
	}

	id, err := c.SendButtons(context.Background(), 42, "выбор", [][]Button{{{Text: "Кафе", Data: "c:1:2"}}})
	if err != nil || id != 55 || gotMethod != "sendMessage" {
		t.Fatalf("send buttons = %d, %v", id, err)
	}
	kb := body["reply_markup"].(map[string]any)["inline_keyboard"].([]any)[0].([]any)[0].(map[string]any)
	if kb["text"] != "Кафе" || kb["callback_data"] != "c:1:2" {
		t.Errorf("keyboard = %v", kb)
	}
}
