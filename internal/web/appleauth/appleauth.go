// Package appleauth проверяет identity token «Входа через Apple»: подпись по открытым ключам Apple, издателя, получателя
// (идентификатор приложения), срок и nonce. Алгоритм: https://developer.apple.com/documentation/signinwithapple
package appleauth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	keysURL   = "https://appleid.apple.com/auth/keys"
	issuer    = "https://appleid.apple.com"
	keysTTL   = time.Hour
	maxReply  = 1 << 20
	clockSkew = time.Minute
)

// ErrInvalid: токен не прошёл проверку. Подробности (какая именно) только в тексте обёрнутой ошибки, наружу не идут.
var ErrInvalid = errors.New("apple identity token invalid")

type Verifier struct {
	clientID string
	keysURL  string
	client   *http.Client
	now      func() time.Time

	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

// New создаёт проверяющего для приложения с идентификатором clientID (bundle id).
func New(clientID string) *Verifier {
	return &Verifier{clientID: clientID, keysURL: keysURL, client: &http.Client{Timeout: 10 * time.Second}, now: time.Now}
}

type header struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type claims struct {
	Iss   string `json:"iss"`
	Aud   string `json:"aud"`
	Sub   string `json:"sub"`
	Exp   int64  `json:"exp"`
	Nonce string `json:"nonce"`
}

// Verify проверяет токен и возвращает идентификатор пользователя Apple (sub). nonce это исходная случайная строка,
// которую приложение выбрало перед входом: в токене лежит её SHA-256 в виде hex. Так украденный токен нельзя
// подсунуть повторно.
func (v *Verifier) Verify(ctx context.Context, identityToken, nonce string) (string, error) {
	parts := strings.Split(identityToken, ".")
	if len(parts) != 3 || nonce == "" {
		return "", fmt.Errorf("%w: malformed", ErrInvalid)
	}
	var h header
	var c claims
	if err := decodePart(parts[0], &h); err != nil {
		return "", fmt.Errorf("%w: header", ErrInvalid)
	}
	if err := decodePart(parts[1], &c); err != nil {
		return "", fmt.Errorf("%w: claims", ErrInvalid)
	}
	// Алгоритм берётся не из токена, а фиксируется: иначе подпись «none» или HMAC обходит проверку.
	if h.Alg != "RS256" || h.Kid == "" {
		return "", fmt.Errorf("%w: alg", ErrInvalid)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("%w: signature", ErrInvalid)
	}
	key, err := v.key(ctx, h.Kid)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
		return "", fmt.Errorf("%w: signature", ErrInvalid)
	}

	if c.Iss != issuer || c.Aud != v.clientID || c.Sub == "" {
		return "", fmt.Errorf("%w: issuer or audience", ErrInvalid)
	}
	if v.now().After(time.Unix(c.Exp, 0).Add(clockSkew)) {
		return "", fmt.Errorf("%w: expired", ErrInvalid)
	}
	sum := sha256.Sum256([]byte(nonce))
	if c.Nonce != hex.EncodeToString(sum[:]) {
		return "", fmt.Errorf("%w: nonce", ErrInvalid)
	}
	return c.Sub, nil
}

func decodePart(part string, out any) error {
	raw, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

// key возвращает открытый ключ по kid. Ключи кэшируются на час; неизвестный kid после свежей загрузки это ошибка токена,
// а не повод ходить в Apple на каждый запрос.
func (v *Verifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if k, ok := v.keys[kid]; ok && v.now().Sub(v.fetched) < keysTTL {
		return k, nil
	}
	if v.keys == nil || v.now().Sub(v.fetched) > time.Minute {
		if err := v.load(ctx); err != nil {
			return nil, err
		}
	}
	k, ok := v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("%w: unknown key", ErrInvalid)
	}
	return k, nil
}

func (v *Verifier) load(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.keysURL, nil)
	if err != nil {
		return fmt.Errorf("apple keys request: %w", err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("apple keys: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apple keys: status %d", resp.StatusCode)
	}
	var set struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxReply)).Decode(&set); err != nil {
		return fmt.Errorf("apple keys: decode: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(k.N)
		e, err2 := base64.RawURLEncoding.DecodeString(k.E)
		if err1 != nil || err2 != nil {
			continue
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	v.keys, v.fetched = keys, v.now()
	return nil
}
