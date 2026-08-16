package asynqmon

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	authSessionCookieName = "asynqmon_session"
	authSessionDuration   = 12 * time.Hour
	loginAttemptWindow    = time.Minute
	maxLoginAttempts      = 5
	maxLoginRequestSize   = 8 << 10
)

var errIncompleteAuthConfig = errors.New("asynqmon: auth username and password must be set together")

type authSession struct {
	expiresAt time.Time
	csrfToken string
}

type loginAttempt struct {
	failures int
	resetAt  time.Time
}

type authenticator struct {
	enabled      bool
	username     string
	usernameHash [sha256.Size]byte
	passwordHash [sha256.Size]byte
	rootPath     string

	mu       sync.Mutex
	sessions map[[sha256.Size]byte]authSession
	attempts map[string]loginAttempt
	now      func() time.Time
}

type authStateResponse struct {
	Enabled       bool   `json:"enabled"`
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
	CSRFToken     string `json:"csrfToken,omitempty"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authErrorResponse struct {
	Error string `json:"error"`
}

func validateAuthConfig(username, password string) error {
	if (username == "") != (password == "") {
		return errIncompleteAuthConfig
	}
	return nil
}

func newAuthenticator(username, password, rootPath string) *authenticator {
	a := &authenticator{
		enabled:  username != "" && password != "",
		username: username,
		rootPath: rootPath,
		sessions: make(map[[sha256.Size]byte]authSession),
		attempts: make(map[string]loginAttempt),
		now:      time.Now,
	}
	if a.enabled {
		a.usernameHash = sha256.Sum256([]byte(username))
		a.passwordHash = sha256.Sum256([]byte(password))
	}
	return a
}

func (a *authenticator) sessionHandler(w http.ResponseWriter, r *http.Request) {
	if !a.enabled {
		writeAuthJSON(w, http.StatusOK, authStateResponse{
			Enabled:       false,
			Authenticated: true,
		})
		return
	}

	session, _, ok := a.sessionFromRequest(r)
	if !ok {
		writeAuthJSON(w, http.StatusOK, authStateResponse{
			Enabled:       true,
			Authenticated: false,
		})
		return
	}

	writeAuthJSON(w, http.StatusOK, authStateResponse{
		Enabled:       true,
		Authenticated: true,
		Username:      a.username,
		CSRFToken:     session.csrfToken,
	})
}

func (a *authenticator) loginHandler(w http.ResponseWriter, r *http.Request) {
	if !a.enabled {
		writeAuthJSON(w, http.StatusOK, authStateResponse{
			Enabled:       false,
			Authenticated: true,
		})
		return
	}

	client := clientAddress(r)
	if allowed, retryAfter := a.loginAllowed(client); !allowed {
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
		writeAuthError(w, http.StatusTooManyRequests, "Too many sign-in attempts. Please try again shortly.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxLoginRequestSize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var credentials loginRequest
	if err := decoder.Decode(&credentials); err != nil {
		writeAuthError(w, http.StatusBadRequest, "Invalid sign-in request.")
		return
	}
	if err := ensureJSONBodyConsumed(decoder); err != nil {
		writeAuthError(w, http.StatusBadRequest, "Invalid sign-in request.")
		return
	}

	if !a.credentialsMatch(credentials.Username, credentials.Password) {
		a.recordLoginFailure(client)
		writeAuthError(w, http.StatusUnauthorized, "Invalid username or password.")
		return
	}

	sessionToken, csrfToken, expiresAt, err := a.createSession()
	if err != nil {
		writeAuthError(w, http.StatusInternalServerError, "Unable to create a session.")
		return
	}
	a.clearLoginFailures(client)
	a.setSessionCookie(w, r, sessionToken, expiresAt)
	writeAuthJSON(w, http.StatusOK, authStateResponse{
		Enabled:       true,
		Authenticated: true,
		Username:      a.username,
		CSRFToken:     csrfToken,
	})
}

func (a *authenticator) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if a.enabled {
		if session, tokenHash, ok := a.sessionFromRequest(r); ok {
			if !csrfTokenMatches(r.Header.Get("X-CSRF-Token"), session.csrfToken) {
				writeAuthError(w, http.StatusForbidden, "The request could not be verified.")
				return
			}
			a.deleteSession(tokenHash)
		}
	}

	a.clearSessionCookie(w, r)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (a *authenticator) requireSession(next http.Handler) http.Handler {
	if !a.enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _, ok := a.sessionFromRequest(r)
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "Authentication is required.")
			return
		}
		if requestRequiresCSRF(r.Method) && !csrfTokenMatches(r.Header.Get("X-CSRF-Token"), session.csrfToken) {
			writeAuthError(w, http.StatusForbidden, "The request could not be verified.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *authenticator) credentialsMatch(username, password string) bool {
	usernameHash := sha256.Sum256([]byte(username))
	passwordHash := sha256.Sum256([]byte(password))
	usernameMatches := subtle.ConstantTimeCompare(usernameHash[:], a.usernameHash[:])
	passwordMatches := subtle.ConstantTimeCompare(passwordHash[:], a.passwordHash[:])
	return usernameMatches&passwordMatches == 1
}

func (a *authenticator) createSession() (sessionToken, csrfToken string, expiresAt time.Time, err error) {
	sessionToken, err = randomToken()
	if err != nil {
		return "", "", time.Time{}, err
	}
	csrfToken, err = randomToken()
	if err != nil {
		return "", "", time.Time{}, err
	}

	expiresAt = a.now().Add(authSessionDuration)
	tokenHash := sha256.Sum256([]byte(sessionToken))
	a.mu.Lock()
	a.sessions[tokenHash] = authSession{
		expiresAt: expiresAt,
		csrfToken: csrfToken,
	}
	a.removeExpiredSessionsLocked(a.now())
	a.mu.Unlock()
	return sessionToken, csrfToken, expiresAt, nil
}

func (a *authenticator) sessionFromRequest(r *http.Request) (authSession, [sha256.Size]byte, bool) {
	var emptyHash [sha256.Size]byte
	cookie, err := r.Cookie(authSessionCookieName)
	if err != nil || cookie.Value == "" {
		return authSession{}, emptyHash, false
	}

	tokenHash := sha256.Sum256([]byte(cookie.Value))
	a.mu.Lock()
	defer a.mu.Unlock()
	session, ok := a.sessions[tokenHash]
	if !ok {
		return authSession{}, tokenHash, false
	}
	if !a.now().Before(session.expiresAt) {
		delete(a.sessions, tokenHash)
		return authSession{}, tokenHash, false
	}
	return session, tokenHash, true
}

func (a *authenticator) deleteSession(tokenHash [sha256.Size]byte) {
	a.mu.Lock()
	delete(a.sessions, tokenHash)
	a.mu.Unlock()
}

func (a *authenticator) removeExpiredSessionsLocked(now time.Time) {
	for tokenHash, session := range a.sessions {
		if !now.Before(session.expiresAt) {
			delete(a.sessions, tokenHash)
		}
	}
}

func (a *authenticator) loginAllowed(client string) (bool, time.Duration) {
	now := a.now()
	a.mu.Lock()
	defer a.mu.Unlock()
	attempt, ok := a.attempts[client]
	if !ok || !now.Before(attempt.resetAt) {
		delete(a.attempts, client)
		return true, 0
	}
	if attempt.failures < maxLoginAttempts {
		return true, 0
	}
	return false, attempt.resetAt.Sub(now)
}

func (a *authenticator) recordLoginFailure(client string) {
	now := a.now()
	a.mu.Lock()
	defer a.mu.Unlock()
	attempt, ok := a.attempts[client]
	if !ok || !now.Before(attempt.resetAt) {
		attempt = loginAttempt{resetAt: now.Add(loginAttemptWindow)}
	}
	attempt.failures++
	a.attempts[client] = attempt

	for address, otherAttempt := range a.attempts {
		if !now.Before(otherAttempt.resetAt) {
			delete(a.attempts, address)
		}
	}
}

func (a *authenticator) clearLoginFailures(client string) {
	a.mu.Lock()
	delete(a.attempts, client)
	a.mu.Unlock()
}

func (a *authenticator) setSessionCookie(w http.ResponseWriter, r *http.Request, value string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     authSessionCookieName,
		Value:    value,
		Path:     a.cookiePath(),
		Expires:  expiresAt,
		MaxAge:   int(authSessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   requestIsSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *authenticator) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authSessionCookieName,
		Path:     a.cookiePath(),
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   requestIsSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *authenticator) cookiePath() string {
	if a.rootPath == "" {
		return "/"
	}
	return a.rootPath
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func ensureJSONBodyConsumed(decoder *json.Decoder) error {
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request body contains multiple JSON values")
		}
		return err
	}
	return nil
}

func requestRequiresCSRF(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func csrfTokenMatches(token, expectedToken string) bool {
	providedHash := sha256.Sum256([]byte(token))
	expectedHash := sha256.Sum256([]byte(expectedToken))
	return token != "" && subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) == 1
}

func requestIsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	forwardedProto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
	return strings.EqualFold(forwardedProto, "https")
}

func clientAddress(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		return strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func writeAuthJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	writeAuthJSON(w, status, authErrorResponse{Error: message})
}
