// Package session implements signed, HTTP-only session cookies via
// gorilla/sessions, carrying only the authenticated user id.
package session

import (
	"net/http"
	"time"

	"github.com/gorilla/sessions"
)

const (
	cookieName = "wow1_session"
	ttl        = 30 * 24 * time.Hour
)

type Manager struct {
	store *sessions.CookieStore
}

func NewManager(secret string) *Manager {
	return &Manager{store: sessions.NewCookieStore([]byte(secret))}
}

func (m *Manager) options(r *http.Request) *sessions.Options {
	return &sessions.Options{
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	}
}

// Set writes a signed session cookie identifying userID.
func (m *Manager) Set(w http.ResponseWriter, r *http.Request, userID int64) error {
	sess, _ := m.store.Get(r, cookieName)
	sess.Options = m.options(r)
	sess.Values["user_id"] = userID
	return sess.Save(r, w)
}

// Clear removes the session cookie.
func (m *Manager) Clear(w http.ResponseWriter, r *http.Request) error {
	sess, _ := m.store.Get(r, cookieName)
	sess.Options = m.options(r)
	sess.Options.MaxAge = -1
	return sess.Save(r, w)
}

// UserID returns the authenticated user id from the request's session
// cookie, or ok=false if there is none or it is invalid/expired.
func (m *Manager) UserID(r *http.Request) (int64, bool) {
	sess, err := m.store.Get(r, cookieName)
	if err != nil {
		return 0, false
	}
	userID, ok := sess.Values["user_id"].(int64)
	return userID, ok
}
