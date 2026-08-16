package asynqmon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

func TestValidateAuthConfig(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{name: "disabled"},
		{name: "enabled", username: "operator", password: "secret"},
		{name: "missing password", username: "operator", wantErr: true},
		{name: "missing username", password: "secret", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAuthConfig(tc.username, tc.password)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateAuthConfig() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestAuthenticationFlow(t *testing.T) {
	auth := newAuthenticator("operator", "correct horse battery staple", "/monitoring")
	protected := auth.requireSession(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	unauthenticated := httptest.NewRecorder()
	protected.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "https://example.com/monitoring/api/queues", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want %d", unauthenticated.Code, http.StatusUnauthorized)
	}

	invalidLogin := httptest.NewRecorder()
	auth.loginHandler(
		invalidLogin,
		httptest.NewRequest(http.MethodPost, "https://example.com/monitoring/api/auth/login", strings.NewReader(`{"username":"operator","password":"wrong"}`)),
	)
	if invalidLogin.Code != http.StatusUnauthorized {
		t.Fatalf("invalid login status = %d, want %d", invalidLogin.Code, http.StatusUnauthorized)
	}

	login := httptest.NewRecorder()
	auth.loginHandler(
		login,
		httptest.NewRequest(http.MethodPost, "https://example.com/monitoring/api/auth/login", strings.NewReader(`{"username":"operator","password":"correct horse battery staple"}`)),
	)
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d: %s", login.Code, http.StatusOK, login.Body.String())
	}
	loginState := decodeAuthState(t, login)
	if !loginState.Enabled || !loginState.Authenticated || loginState.Username != "operator" || loginState.CSRFToken == "" {
		t.Fatalf("unexpected login state: %+v", loginState)
	}

	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("login returned %d cookies, want 1", len(cookies))
	}
	sessionCookie := cookies[0]
	if sessionCookie.Name != authSessionCookieName || sessionCookie.Path != "/monitoring" || !sessionCookie.HttpOnly || !sessionCookie.Secure || sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected session cookie: %+v", sessionCookie)
	}

	sessionRequest := httptest.NewRequest(http.MethodGet, "https://example.com/monitoring/api/auth/session", nil)
	sessionRequest.AddCookie(sessionCookie)
	sessionRecorder := httptest.NewRecorder()
	auth.sessionHandler(sessionRecorder, sessionRequest)
	sessionState := decodeAuthState(t, sessionRecorder)
	if !sessionState.Authenticated || sessionState.CSRFToken != loginState.CSRFToken {
		t.Fatalf("unexpected session state: %+v", sessionState)
	}

	protectedGet := httptest.NewRequest(http.MethodGet, "https://example.com/monitoring/api/queues", nil)
	protectedGet.AddCookie(sessionCookie)
	protectedGetRecorder := httptest.NewRecorder()
	protected.ServeHTTP(protectedGetRecorder, protectedGet)
	if protectedGetRecorder.Code != http.StatusNoContent {
		t.Fatalf("authenticated GET status = %d, want %d", protectedGetRecorder.Code, http.StatusNoContent)
	}

	missingCSRF := httptest.NewRequest(http.MethodPost, "https://example.com/monitoring/api/queues/default:pause", nil)
	missingCSRF.AddCookie(sessionCookie)
	missingCSRFRecorder := httptest.NewRecorder()
	protected.ServeHTTP(missingCSRFRecorder, missingCSRF)
	if missingCSRFRecorder.Code != http.StatusForbidden {
		t.Fatalf("POST without CSRF status = %d, want %d", missingCSRFRecorder.Code, http.StatusForbidden)
	}

	protectedPost := httptest.NewRequest(http.MethodPost, "https://example.com/monitoring/api/queues/default:pause", nil)
	protectedPost.AddCookie(sessionCookie)
	protectedPost.Header.Set("X-CSRF-Token", loginState.CSRFToken)
	protectedPostRecorder := httptest.NewRecorder()
	protected.ServeHTTP(protectedPostRecorder, protectedPost)
	if protectedPostRecorder.Code != http.StatusNoContent {
		t.Fatalf("authenticated POST status = %d, want %d", protectedPostRecorder.Code, http.StatusNoContent)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "https://example.com/monitoring/api/auth/logout", nil)
	logoutRequest.AddCookie(sessionCookie)
	logoutRequest.Header.Set("X-CSRF-Token", loginState.CSRFToken)
	logoutRecorder := httptest.NewRecorder()
	auth.logoutHandler(logoutRecorder, logoutRequest)
	if logoutRecorder.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want %d", logoutRecorder.Code, http.StatusNoContent)
	}

	afterLogoutRequest := httptest.NewRequest(http.MethodGet, "https://example.com/monitoring/api/auth/session", nil)
	afterLogoutRequest.AddCookie(sessionCookie)
	afterLogoutRecorder := httptest.NewRecorder()
	auth.sessionHandler(afterLogoutRecorder, afterLogoutRequest)
	if state := decodeAuthState(t, afterLogoutRecorder); state.Authenticated {
		t.Fatalf("session remained authenticated after logout: %+v", state)
	}
}

func TestAuthenticationRateLimitAndExpiry(t *testing.T) {
	auth := newAuthenticator("operator", "secret", "")
	now := time.Date(2026, time.August, 16, 12, 0, 0, 0, time.UTC)
	auth.now = func() time.Time { return now }

	for attempt := 1; attempt <= maxLoginAttempts; attempt++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "http://example.com/api/auth/login", strings.NewReader(`{"username":"operator","password":"wrong"}`))
		request.RemoteAddr = "203.0.113.10:1234"
		auth.loginHandler(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("failed login %d status = %d, want %d", attempt, recorder.Code, http.StatusUnauthorized)
		}
	}

	limited := httptest.NewRecorder()
	limitedRequest := httptest.NewRequest(http.MethodPost, "http://example.com/api/auth/login", strings.NewReader(`{"username":"operator","password":"secret"}`))
	limitedRequest.RemoteAddr = "203.0.113.10:1234"
	auth.loginHandler(limited, limitedRequest)
	if limited.Code != http.StatusTooManyRequests || limited.Header().Get("Retry-After") == "" {
		t.Fatalf("rate-limited login = %d, Retry-After %q", limited.Code, limited.Header().Get("Retry-After"))
	}

	now = now.Add(loginAttemptWindow)
	login := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "http://example.com/api/auth/login", strings.NewReader(`{"username":"operator","password":"secret"}`))
	loginRequest.RemoteAddr = "203.0.113.10:1234"
	auth.loginHandler(login, loginRequest)
	if login.Code != http.StatusOK {
		t.Fatalf("login after rate-limit window status = %d, want %d", login.Code, http.StatusOK)
	}
	sessionCookie := login.Result().Cookies()[0]

	now = now.Add(authSessionDuration)
	protected := auth.requireSession(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	expiredRequest := httptest.NewRequest(http.MethodGet, "http://example.com/api/queues", nil)
	expiredRequest.AddCookie(sessionCookie)
	expiredRecorder := httptest.NewRecorder()
	protected.ServeHTTP(expiredRecorder, expiredRequest)
	if expiredRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status = %d, want %d", expiredRecorder.Code, http.StatusUnauthorized)
	}
}

func TestSessionCookieIsSecureBehindHTTPSProxy(t *testing.T) {
	auth := newAuthenticator("operator", "secret", "")
	request := httptest.NewRequest(
		http.MethodPost,
		"http://example.com/api/auth/login",
		strings.NewReader(`{"username":"operator","password":"secret"}`),
	)
	request.Header.Set("X-Forwarded-Proto", "https")
	recorder := httptest.NewRecorder()

	auth.loginHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", recorder.Code, http.StatusOK)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure {
		t.Fatalf("session cookie behind HTTPS proxy was not secure: %+v", cookies)
	}
}

func TestHTTPHandlerAuthRoutesRespectRootPath(t *testing.T) {
	h := New(Options{
		RootPath:     "/monitoring",
		RedisConnOpt: asynq.RedisClientOpt{Addr: "127.0.0.1:6379"},
		AuthUsername: "operator",
		AuthPassword: "secret",
	})
	defer h.Close()

	sessionRecorder := httptest.NewRecorder()
	h.ServeHTTP(sessionRecorder, httptest.NewRequest(http.MethodGet, "http://example.com/monitoring/api/auth/session", nil))
	if sessionRecorder.Code != http.StatusOK {
		t.Fatalf("session endpoint status = %d, want %d", sessionRecorder.Code, http.StatusOK)
	}
	state := decodeAuthState(t, sessionRecorder)
	if !state.Enabled || state.Authenticated {
		t.Fatalf("unexpected initial auth state: %+v", state)
	}

	queuesRecorder := httptest.NewRecorder()
	h.ServeHTTP(queuesRecorder, httptest.NewRequest(http.MethodGet, "http://example.com/monitoring/api/queues", nil))
	if queuesRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("protected API status = %d, want %d", queuesRecorder.Code, http.StatusUnauthorized)
	}
}

func TestDisabledAuthenticationAllowsAPIs(t *testing.T) {
	auth := newAuthenticator("", "", "")
	protected := auth.requireSession(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	protected.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "http://example.com/api/queues/default", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("disabled auth status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func decodeAuthState(t *testing.T, recorder *httptest.ResponseRecorder) authStateResponse {
	t.Helper()
	var state authStateResponse
	if err := json.NewDecoder(recorder.Body).Decode(&state); err != nil {
		t.Fatalf("decode auth state: %v", err)
	}
	return state
}
