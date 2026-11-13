package appleauth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var testNow = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

const (
	testClient = "com.example.moneykeeper"
	testNonce  = "raw-nonce-123"
)

func nonceHash(n string) string {
	sum := sha256.Sum256([]byte(n))
	return hex.EncodeToString(sum[:])
}

func b64(v any) string {
	raw, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(raw)
}

// token собирает подписанный identity token; claims переопределяют стандартные поля.
func token(t *testing.T, key *rsa.PrivateKey, kid, alg string, override map[string]any) string {
	t.Helper()
	claims := map[string]any{"iss": issuer, "aud": testClient, "sub": "001.apple.user", "exp": testNow.Add(10 * time.Minute).Unix(), "nonce": nonceHash(testNonce)}
	for k, v := range override {
		claims[k] = v
	}
	signing := b64(map[string]string{"alg": alg, "kid": kid}) + "." + b64(claims)
	digest := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func newVerifier(t *testing.T, key *rsa.PrivateKey, kid string, hits *int) *Verifier {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits != nil {
			*hits++
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	t.Cleanup(srv.Close)
	v := New(testClient)
	v.keysURL, v.client, v.now = srv.URL, srv.Client(), func() time.Time { return testNow }
	return v
}

func TestVerify(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	v := newVerifier(t, key, "k1", nil)

	tests := []struct {
		name  string
		token string
		nonce string
		want  bool
	}{
		{"valid", token(t, key, "k1", "RS256", nil), testNonce, true},
		{"just expired within skew", token(t, key, "k1", "RS256", map[string]any{"exp": testNow.Add(-30 * time.Second).Unix()}), testNonce, true},
		{"expired", token(t, key, "k1", "RS256", map[string]any{"exp": testNow.Add(-5 * time.Minute).Unix()}), testNonce, false},
		{"other app", token(t, key, "k1", "RS256", map[string]any{"aud": "com.other.app"}), testNonce, false},
		{"other issuer", token(t, key, "k1", "RS256", map[string]any{"iss": "https://evil.example"}), testNonce, false},
		{"no subject", token(t, key, "k1", "RS256", map[string]any{"sub": ""}), testNonce, false},
		{"wrong nonce", token(t, key, "k1", "RS256", nil), "another-nonce", false},
		{"empty nonce", token(t, key, "k1", "RS256", nil), "", false},
		{"signed with another key", token(t, other, "k1", "RS256", nil), testNonce, false},
		{"unknown key id", token(t, key, "k2", "RS256", nil), testNonce, false},
		{"alg none", token(t, key, "k1", "none", nil), testNonce, false},
		{"alg HS256", token(t, key, "k1", "HS256", nil), testNonce, false},
		{"garbage", "not.a.jwt", testNonce, false},
		{"two parts", "a.b", testNonce, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, err := v.Verify(context.Background(), tt.token, tt.nonce)
			if tt.want {
				if err != nil || sub != "001.apple.user" {
					t.Fatalf("got %q, %v; want the subject", sub, err)
				}
				return
			}
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("err = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestKeysAreCached(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	hits := 0
	v := newVerifier(t, key, "k1", &hits)
	tok := token(t, key, "k1", "RS256", nil)
	for range 3 {
		if _, err := v.Verify(context.Background(), tok, testNonce); err != nil {
			t.Fatal(err)
		}
	}
	if hits != 1 {
		t.Errorf("key requests = %d, want 1", hits)
	}
}
