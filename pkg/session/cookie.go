package session

import (
	"net/http"
	"time"
)

func (s *SessionSubsystem) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete the cookie
		Secure:   s.config.SessionCookieSecure,
		HttpOnly: s.config.SessionCookieHttpOnly,
		SameSite: http.SameSiteLaxMode, // Use Lax for clearing
	})
}

func (s *SessionSubsystem) setSessionCookie(w http.ResponseWriter, sessionID string, expiresAt time.Time) {
	var sameSite http.SameSite
	switch s.config.SessionCookieSameSite {
	case "none":
		sameSite = http.SameSiteNoneMode
	case "default":
		sameSite = http.SameSiteDefaultMode
	case "lax":
		sameSite = http.SameSiteLaxMode
	case "strict":
		sameSite = http.SameSiteStrictMode
	default:
		sameSite = http.SameSiteLaxMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.config.SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  expiresAt,
		Secure:   s.config.SessionCookieSecure,
		HttpOnly: s.config.SessionCookieHttpOnly,
		SameSite: sameSite,
	})
}

const state_cookie_name = "state"

func setStateCookie(w http.ResponseWriter, value string, secure bool) {
	c := &http.Cookie{
		Name:     state_cookie_name,
		Value:    value,
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   secure,
		HttpOnly: true,
	}
	http.SetCookie(w, c)
}

func setCodeVerifierCookie(w http.ResponseWriter, codeVerifier string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "code_verifier",
		Value:    codeVerifier,
		MaxAge:   600, // 10 minutes
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
