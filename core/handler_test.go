package gkit

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestLogErrorHandlerUsesSlog(t *testing.T) {
	var buf bytes.Buffer

	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	LogErrorHandler(nil)(context.Background(), errors.New("dang"))

	out := buf.String()
	if !strings.Contains(out, "request failed") {
		t.Errorf("log message missing: %q", out)
	}
	if !strings.Contains(out, "error=dang") {
		t.Errorf("error attribute missing: %q", out)
	}
}

func TestLogErrorHandlerUsesLogFunc(t *testing.T) {
	var got error

	LogErrorHandler(func(_ context.Context, err error) { got = err })(context.Background(), errors.New("dang"))

	if got == nil || got.Error() != "dang" {
		t.Errorf("want %q, have %v", "dang", got)
	}
}
