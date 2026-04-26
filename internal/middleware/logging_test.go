package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beastixq/marketplace/internal/middleware"
)

func TestRecoverer_LogsPanicAndRequestLoggerRecords500(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})
	wrapped := middleware.RequestLogger(logger)(middleware.Recoverer(logger)(handler))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	out := logs.String()
	for _, want := range []string{
		`"msg":"http panic recovered"`,
		`"panic":"boom"`,
		`"msg":"http request"`,
		`"status":500`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("logs missing %s:\n%s", want, out)
		}
	}
}

func TestRequestLogger_RecordsFirstStatus(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.WriteHeader(http.StatusInternalServerError)
	})
	wrapped := middleware.RequestLogger(logger)(handler)

	wrapped.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/status", nil))

	if out := logs.String(); !strings.Contains(out, `"status":400`) {
		t.Fatalf("logs should contain first status 400:\n%s", out)
	}
}
