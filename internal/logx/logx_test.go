package logx_test

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"github.com/DeanXu2357/taskbe/internal/logx"
)

func TestStdLoggerQuotesValues(t *testing.T) {
	var buf bytes.Buffer
	logger := logx.New(log.New(&buf, "", 0))

	logger.Info("request",
		"path", "/tasks/1\n2026/11/17 19:24:08 ERROR forged",
		"method", "GET",
		"status", 418,
		"error", errors.New("get task 1: boom"),
		"empty", "",
	)

	want := `INFO request path "/tasks/1\n2026/11/17 19:24:08 ERROR forged" method GET status 418 error "get task 1: boom" empty ""` + "\n"
	if got := buf.String(); got != want {
		t.Errorf("log line:\n got %q\nwant %q", got, want)
	}
}
