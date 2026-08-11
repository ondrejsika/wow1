// Package oidcauth implements login via a generic OpenID Connect provider
// using the authorization code flow, with state and nonce verification.
package oidcauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"wow1/internal/session"
	"wow1/internal/store"
)

const (
	stateCookieName = "wow1_oidc_state"
	nonceCookieName = "wow1_oidc_nonce"
	oidcCookieTTL   = 10 * time.Minute
)

type Service struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
	store        *store.Store
	sessions     *session.Manager
}

func New(ctx context.Context, issuerURL, clientID, clientSecret, redirectURL string, st *store.Store, sessions *session.Manager) (*Service, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	return &Service{
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
		oauth2Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		store:    st,
		sessions: sessions,
	}, nil
}

// Login starts the authorization code flow: it generates state and nonce,
// stores them in short-lived cookies, and redirects to the provider.
func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	state, err := randomToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	nonce, err := randomToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	setOIDCCookie(w, r, stateCookieName, state)
	setOIDCCookie(w, r, nonceCookieName, nonce)

	http.Redirect(w, r, s.oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
}

// Callback completes the authorization code flow: it verifies state and
// nonce, exchanges the code, verifies the ID token, and upserts the user
// before establishing a session.
func (s *Service) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || !session.ConstantTimeEqual(r.URL.Query().Get("state"), stateCookie.Value) {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	nonceCookie, err := r.Cookie(nonceCookieName)
	if err != nil {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	clearOIDCCookie(w, r, stateCookieName)
	clearOIDCCookie(w, r, nonceCookieName)

	oauth2Token, err := s.oauth2Config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("oidc: code exchange failed: %v", err)
		http.Error(w, "authentication failed", http.StatusBadGateway)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "authentication failed: no id_token", http.StatusBadGateway)
		return
	}
	idToken, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("oidc: id_token verification failed: %v", err)
		http.Error(w, "authentication failed", http.StatusBadGateway)
		return
	}
	if !session.ConstantTimeEqual(idToken.Nonce, nonceCookie.Value) {
		http.Error(w, "invalid nonce", http.StatusBadRequest)
		return
	}

	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "authentication failed: bad claims", http.StatusBadGateway)
		return
	}
	if claims.Name == "" {
		claims.Name = claims.Email
	}

	user, err := s.store.UpsertUser(ctx, idToken.Subject, claims.Email, claims.Name)
	if err != nil {
		log.Printf("oidc: upsert user failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.sessions.Set(w, r, user.ID)
	http.Redirect(w, r, "/", http.StatusFound)
}

// Logout clears the session cookie.
func Logout(sessions *session.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions.Clear(w, r)
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setOIDCCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(oidcCookieTTL),
	})
}

func clearOIDCCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
