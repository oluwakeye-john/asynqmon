package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorLoggingMiddlewareLogsServerErrors(t *testing.T) {
	var output bytes.Buffer
	logger := log.New(&output, "", 0)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "dial tcp 127.0.0.1:6379: connect: connection refused", http.StatusInternalServerError)
	})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/queues?token=do-not-log", nil)
	req.RemoteAddr = "10.0.0.7:43120"

	errorLoggingMiddleware(logger, handler).ServeHTTP(httptest.NewRecorder(), req)

	logLine := output.String()
	wants := []string{
		`level=ERROR`,
		`component=http`,
		`method="GET"`,
		`path="/api/queues"`,
		`status=500`,
		`remote_addr="10.0.0.7:43120"`,
		`error="dial tcp 127.0.0.1:6379: connect: connection refused"`,
	}
	for _, want := range wants {
		if !strings.Contains(logLine, want) {
			t.Errorf("log line %q does not contain %q", logLine, want)
		}
	}
	if strings.Contains(logLine, "do-not-log") {
		t.Errorf("log line includes sensitive query string: %q", logLine)
	}
}

func TestErrorLoggingMiddlewareIgnoresNonServerErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{name: "success", status: http.StatusOK},
		{name: "client error", status: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := log.New(&output, "", 0)
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			})

			errorLoggingMiddleware(logger, handler).ServeHTTP(
				httptest.NewRecorder(),
				httptest.NewRequest(http.MethodGet, "http://example.com/api/queues", nil),
			)

			if output.Len() != 0 {
				t.Errorf("unexpected log output for status %d: %q", tc.status, output.String())
			}
		})
	}
}

func TestErrorLoggingMiddlewareBoundsErrorMessage(t *testing.T) {
	var output bytes.Buffer
	logger := log.New(&output, "", 0)
	errorBody := strings.Repeat("x", maxLoggedErrorBodyBytes+100)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, errorBody, http.StatusInternalServerError)
	})

	errorLoggingMiddleware(logger, handler).ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "http://example.com/api/queues", nil),
	)

	if !strings.Contains(output.String(), "[truncated]") {
		t.Fatalf("expected bounded error log to be marked as truncated: %q", output.String())
	}
	if output.Len() > maxLoggedErrorBodyBytes+500 {
		t.Fatalf("error log is unexpectedly large: got %d bytes", output.Len())
	}
}
