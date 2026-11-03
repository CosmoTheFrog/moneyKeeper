// Package pdftext достаёт текст из PDF внешней программой pdftotext (poppler) с сохранением колонок.
package pdftext

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

const timeout = 20 * time.Second

var (
	// ErrUnavailable: pdftotext не установлен.
	ErrUnavailable = errors.New("pdftotext is not installed")
	ErrInvalid     = errors.New("cannot read pdf")
)

// Extract возвращает текст всех страниц. Файл идёт в программу через stdin, на диск не пишется.
func Extract(ctx context.Context, pdf []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrUnavailable
		}
		return "", fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	return out.String(), nil
}
