package main

import (
	"bytes"
	"log"
	"net/http"
	"strings"
	"time"
)

const maxLoggedErrorBodyBytes = 2048

// A responseRecorderWriter records response status and size.
// It implements http.ResponseWriter interface.
type responseRecorderWriter struct {
	http.ResponseWriter
	// The status code that the server sends back to the client.
	status int
	// The size of the object returned to the client, not including the response headers.
	size int
	// A bounded copy of a server-error response body for container logs.
	errorBody          bytes.Buffer
	errorBodyTruncated bool
}

func (w *responseRecorderWriter) WriteHeader(status int) {
	// net/http ignores all WriteHeader calls after the first one. Mirror that
	// behavior so the recorded status matches the response sent to the client.
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorderWriter) Write(b []byte) (int, error) {
	// If WriteHeader is not called explicitly, the first call to Write
	// will trigger an implicit WriteHeader(http.StatusOK).
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	if w.status >= http.StatusInternalServerError && n > 0 {
		remaining := maxLoggedErrorBodyBytes - w.errorBody.Len()
		if remaining > 0 {
			captured := n
			if captured > remaining {
				captured = remaining
			}
			_, _ = w.errorBody.Write(b[:captured])
		}
		if n > remaining {
			w.errorBodyTruncated = true
		}
	}
	return n, err
}

// Unwrap lets net/http helpers access optional interfaces implemented by the
// original ResponseWriter.
func (w *responseRecorderWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseRecorderWriter) errorMessage() string {
	message := strings.TrimSpace(w.errorBody.String())
	if message == "" {
		message = http.StatusText(w.status)
	}
	if w.errorBodyTruncated {
		message += " [truncated]"
	}
	return message
}

// errorLoggingMiddleware writes failed backend requests to the Go logger,
// which makes Redis and other API failures visible in container logs. Query
// strings are intentionally excluded because they may contain sensitive data.
func errorLoggingMiddleware(logger *log.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseRecorderWriter{ResponseWriter: w}
		startedAt := time.Now()
		h.ServeHTTP(rw, r)

		if rw.status == 0 {
			rw.status = http.StatusOK
		}
		if rw.status < http.StatusInternalServerError {
			return
		}

		logger.Printf(
			"level=ERROR component=http msg=%q method=%q path=%q status=%d bytes=%d duration=%q remote_addr=%q error=%q",
			"HTTP request failed",
			r.Method,
			r.URL.Path,
			rw.status,
			rw.size,
			time.Since(startedAt).String(),
			r.RemoteAddr,
			rw.errorMessage(),
		)
	})
}
