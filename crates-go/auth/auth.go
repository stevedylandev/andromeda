// Package auth provides session storage, password verification and small
// helpers shared by andromeda Go apps.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const shortIDAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

// Store manages session tokens in a sessions table inside the given DB.
type Store struct {
	DB           *sql.DB
	CookieName   string
	CookieSecure bool
	// MaxAge is the cookie lifetime. Defaults to 7 days when zero.
	MaxAge time.Duration
}

// EnsureSchema creates the sessions table if it does not exist.
func (s *Store) EnsureSchema() error {
	_, err := s.DB.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		token      TEXT NOT NULL UNIQUE,
		expires_at TEXT NOT NULL
	)`)
	return err
}

// Create issues a new session, persists it, and returns the raw token.
func (s *Store) Create() (string, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}
	expires := time.Now().UTC().Add(s.maxAge())
	if _, err := s.DB.Exec(`INSERT INTO sessions (token, expires_at) VALUES (?, ?)`,
		token, expires.Format("2006-01-02 15:04:05")); err != nil {
		return "", err
	}
	return token, nil
}

// Valid reports whether the given token exists and has not expired.
func (s *Store) Valid(token string) bool {
	if token == "" {
		return false
	}
	var expires string
	err := s.DB.QueryRow(`SELECT expires_at FROM sessions WHERE token = ?`, token).Scan(&expires)
	if err != nil {
		return false
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", expires, time.UTC)
	return err == nil && t.After(time.Now().UTC())
}

// Delete removes the given session token if present.
func (s *Store) Delete(token string) {
	_, _ = s.DB.Exec(`DELETE FROM sessions WHERE token = ?`, token)
}

// PruneExpired removes all expired session rows.
func (s *Store) PruneExpired() {
	_, _ = s.DB.Exec(`DELETE FROM sessions WHERE expires_at < datetime('now')`)
}

// HasValid checks the cookie on r and returns true if it carries a live session.
func (s *Store) HasValid(r *http.Request) bool {
	c, err := r.Cookie(s.CookieName)
	if err != nil || c.Value == "" {
		return false
	}
	return s.Valid(c.Value)
}

// RequireSession wraps next and redirects to redirectPath when no valid session
// cookie is present.
func (s *Store) RequireSession(redirectPath string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.HasValid(r) {
			http.Redirect(w, r, redirectPath, http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// SessionCookie returns a cookie configured with the store's name, security
// flag and MaxAge.
func (s *Store) SessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     s.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.maxAge().Seconds()),
	}
}

// ClearCookie returns an expired cookie used to log out a session.
func (s *Store) ClearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     s.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
}

func (s *Store) maxAge() time.Duration {
	if s.MaxAge > 0 {
		return s.MaxAge
	}
	return 7 * 24 * time.Hour
}

// RequireAPIKey wraps next with a strict X-API-Key header check.
// Returns 403 if expected is empty, 401 on mismatch.
func RequireAPIKey(expected string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if expected == "" {
			http.Error(w, "API key not configured on server", http.StatusForbidden)
			return
		}
		if !SecureEqual(r.Header.Get("x-api-key"), expected) {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// RequireBearerOrSession allows requests carrying either a matching bearer
// token or a valid session cookie. Falls through with 401 JSON otherwise.
func RequireBearerOrSession(store *Store, expectedBearer string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if expectedBearer != "" {
			authz := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") &&
				SecureEqual(strings.TrimSpace(authz[7:]), expectedBearer) {
				next(w, r)
				return
			}
		}
		if store != nil && store.HasValid(r) {
			next(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}
}

// VerifyPassword checks input against expected. If expected looks like a bcrypt
// hash (`$2`-prefixed) it is compared as such, otherwise as plain text.
func VerifyPassword(input, expected string) bool {
	if strings.HasPrefix(expected, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(expected), []byte(input)) == nil
	}
	return SecureEqual(input, expected)
}

// SecureEqual reports whether a and b are equal in constant time.
func SecureEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// GenerateSessionToken returns a 32-byte random hex token.
func GenerateSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// GenerateShortID returns an n-character URL-safe identifier.
func GenerateShortID(n int) (string, error) {
	if n <= 0 {
		return "", errors.New("auth: short id length must be positive")
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = shortIDAlphabet[int(b)%len(shortIDAlphabet)]
	}
	return string(out), nil
}
