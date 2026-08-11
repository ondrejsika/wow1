// Package oidcauth implements login via a generic OpenID Connect provider
// using zitadel/oidc's relying-party helpers, which handle the
// authorization code flow's state/cookie/token-exchange/verification
// mechanics.
package oidcauth

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"

	"github.com/zitadel/oidc/v3/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	"wow1/internal/session"
	"wow1/internal/store"
)

type Service struct {
	relyingParty rp.RelyingParty
	store        *store.Store
	sessions     *session.Manager
}

func New(ctx context.Context, issuerURL, clientID, clientSecret, redirectURL string, st *store.Store, sessions *session.Manager) (*Service, error) {
	hashKey, encryptKey := make([]byte, 32), make([]byte, 32)
	if _, err := rand.Read(hashKey); err != nil {
		return nil, fmt.Errorf("generate cookie hash key: %w", err)
	}
	if _, err := rand.Read(encryptKey); err != nil {
		return nil, fmt.Errorf("generate cookie encrypt key: %w", err)
	}
	cookieHandler := httphelper.NewCookieHandler(hashKey, encryptKey, httphelper.WithUnsecure())

	relyingParty, err := rp.NewRelyingPartyOIDC(ctx, issuerURL, clientID, clientSecret, redirectURL,
		[]string{oidc.ScopeOpenID, "profile", "email"},
		rp.WithCookieHandler(cookieHandler),
	)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	return &Service{relyingParty: relyingParty, store: st, sessions: sessions}, nil
}

// Login redirects to the provider's authorization endpoint, with state
// transferred via a short-lived cookie.
func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	rp.AuthURLHandler(randomState, s.relyingParty)(w, r)
}

// Callback completes the authorization code flow, upserts the user from
// the verified ID token claims, and establishes a session.
func (s *Service) Callback(w http.ResponseWriter, r *http.Request) {
	rp.CodeExchangeHandler(rp.CodeExchangeCallback[*oidc.IDTokenClaims](s.exchanged), s.relyingParty)(w, r)
}

func (s *Service) exchanged(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], _ string, _ rp.RelyingParty) {
	claims := tokens.IDTokenClaims
	name := claims.Name
	if name == "" {
		name = claims.Email
	}

	user, err := s.store.UpsertUser(r.Context(), claims.Subject, claims.Email, name)
	if err != nil {
		log.Printf("oidc: upsert user failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.sessions.Set(w, r, user.ID); err != nil {
		log.Printf("oidc: set session failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// Logout clears the session cookie.
func Logout(sessions *session.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = sessions.Clear(w, r)
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
