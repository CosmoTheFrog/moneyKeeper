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
