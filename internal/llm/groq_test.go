package llm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGroqCompleteJSON(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    string
		wantErr bool
	}{
		{"ok", http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"{\"rows\":[]}"}}]}`, `{"rows":[]}`, false},
		{"rate limit", http.StatusTooManyRequests, `{"error":"slow down"}`, "", true},
		{"no choices", http.StatusOK, `{"choices":[]}`, "", true},
		{"garbage", http.StatusOK, `not json`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer key" || !strings.HasSuffix(r.URL.Path, "/chat/completions") {
					t.Errorf("unexpected request: %s %s auth=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			g := NewGroq("key")
			g.base = srv.URL
			got, err := g.CompleteJSON(context.Background(), "sys", "user")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrUnavailable) {
				t.Errorf("err = %v, want ErrUnavailable", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
