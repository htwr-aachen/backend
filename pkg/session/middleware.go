package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/htwr-aachen/backend/pkg/schema"
	"github.com/rs/zerolog/log"
)

type contextKey string

const userContextKey contextKey = "user"

// GetUserFromContext retrieves the User from the request context.
func GetUserFromContext(ctx context.Context) (*schema.User, bool) {
	user, ok := ctx.Value(userContextKey).(*schema.User)
	return user, ok
}

func (s *SessionSubsystem) AuthMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == s.config.AuthLoginURL {
			log.Trace().Str("path", r.URL.Path).Msg("auth middleware: bypassing auth for login URL")
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, s.config.AuthURLPrefix) {
			log.Trace().Str("path", r.URL.Path).Str("prefix", s.config.AuthURLPrefix).Msg("auth middleware: routing to auth handler")
			s.mux.ServeHTTP(w, r)
			return
		}

		sessionCookie, err := r.Cookie(s.config.SessionCookieName)
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				// No session cookie, redirect to login
				log.Debug().
					Str("cookie_name", s.config.SessionCookieName).
					Msg("auth middleware: no session cookie found, redirecting to login")
				http.Redirect(w, r, s.getLoginURL(), http.StatusFound)
				return
			}
			log.Err(err).Msg("failed to read session cookie")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		user, err := s.db.LookupSession(r.Context(), sessionCookie.Value)
		if err != nil {
			if !errors.Is(err, schema.NoSessionError{}) {
				log.Debug().
					Str("session_id", sessionCookie.Value[:4]+"...").
					Msg("auth middleware: session lookup error, redirecting to login")
				log.Err(err).Msg("failed to lookup session")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			loginURL := s.getLoginURL()
			log.Debug().
				Str("session_id", sessionCookie.Value[:4]+"...").
				Str("login_url", loginURL).
				Msg("auth middleware: session not found or expired, redirecting to login")
			// Session not found or expired, clear cookie and redirect to login
			s.clearSessionCookie(w)

			http.Redirect(w, r, s.getLoginURL(), http.StatusFound)
			return
		}

		log.Debug().
			Str("path", r.URL.Path).
			Str("user_id", user.Id).
			Str("username", user.Username).
			Str("role", string(user.Role)).
			Msg("auth middleware: session valid, user authenticated")

		// Add user to context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *SessionSubsystem) RequestUser(r *http.Request) (*schema.User, error) {
	user, ok := r.Context().Value(userContextKey).(*schema.User)
	if !ok || user == nil {
		return nil, schema.Unauthenticated{}
	}

	return user, nil
}

func (s *SessionSubsystem) getLoginURL() string {
	// If there's exactly one provider, redirect directly to it
	if len(s.config.Providers) == 1 {
		for providerName := range s.config.Providers {
			log.Debug().Str("provider", providerName).Msg("using as default provider")
			return fmt.Sprintf("%s/%s/login", s.config.AuthURLPrefix, providerName)
		}
	}

	if s.config.AuthLoginURL != "" {
		return s.config.AuthLoginURL
	}

	return fmt.Sprintf("%s/login", s.config.AuthURLPrefix)
}
