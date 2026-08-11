// Package session implements signed, HTTP-only session cookies with no
// server-side session store: the cookie itself carries the user id and an
// expiry, authenticated with an HMAC over SESSION_SECRET.
package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	CookieName = "wow1_session"
	ttl        = 30 * 24 * time.Hour
)

type Manager struct {
	secret []byte
}

func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

// Set writes a signed session cookie identifying userID.
func (m *Manager) Set(w http.ResponseWriter, r *http.Request, userID int64) {
	expires := time.Now().Add(ttl)
	payload := fmt.Sprintf("%d.%d", userID, expires.Unix())
	sig := m.sign(payload)
	value := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(sig)

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	})
}

// Clear removes the session cookie.
func (m *Manager) Clear(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// UserID returns the authenticated user id from the request's session
// cookie, or ok=false if there is none or it is invalid/expired.
func (m *Manager) UserID(r *http.Request) (int64, bool) {
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return 0, false
	}

	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return 0, false
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, false
	}
	if !hmac.Equal(sig, m.sign(string(payloadBytes))) {
		return 0, false
	}

	payloadParts := strings.SplitN(string(payloadBytes), ".", 2)
	if len(payloadParts) != 2 {
		return 0, false
	}
	userID, err := strconv.ParseInt(payloadParts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	expiresUnix, err := strconv.ParseInt(payloadParts[1], 10, 64)
	if err != nil {
		return 0, false
	}
	if time.Now().Unix() > expiresUnix {
		return 0, false
	}

	return userID, true
}

func (m *Manager) sign(payload string) []byte {
	h := hmac.New(sha256.New, m.secret)
	h.Write([]byte(payload))
	return h.Sum(nil)
}

// ConstantTimeEqual is a small helper kept for callers that need to compare
// tokens (e.g. OIDC state) without leaking timing information.
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
