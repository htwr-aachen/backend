package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/htwr-aachen/backend/pkg/schema"
	"github.com/rs/zerolog/log"
)

func (s *SessionSubsystem) newSession(w http.ResponseWriter, r *http.Request, claims UserClaims, identity_provider string) {
	// 1. Lookup or Create User
	user, err := s.db.GetUserByIssSubHash(r.Context(), claims.Iss, claims.Sub)
	if err != nil {
		if errors.Is(err, schema.NoUserError{}) {
			newUser := &schema.User{
				IssSubHash:       getIssSubHash(claims.Iss, claims.Sub),
				Username:         claims.PreferredUsername,
				Name:             claims.Name,
				Email:            claims.Email,
				AvatarURL:        claims.AvatarURL,
				IdentityProvider: identity_provider,
				Role:             s.GetRoleFromGroups(claims.Groups),
			}
			user, err = s.db.CreateUser(r.Context(), newUser)
			if err != nil {
				log.Err(err).Msg("failed to create new user")
				http.Error(w, "failed to create user", http.StatusInternalServerError)
				return
			}
		} else {
			log.Err(err).Str("issuer", claims.Iss).Str("subject", claims.Sub).Msg("failed to lookup user by issuer subject hash")
			http.Error(w, "failed to lookup user", http.StatusInternalServerError)
			return
		}
	}

	// 2. Create session
	sessionID := generateSessionID()
	expiresAt := time.Now().Add(s.config.SessionExpiration)

	// Extract IP address without port
	ipAddress := r.RemoteAddr
	if host, _, err := net.SplitHostPort(ipAddress); err == nil {
		ipAddress = host
	}

	sess := &schema.Session{
		SessionId:    sessionID,
		UserId:       user.Id,
		CreatedAt:    time.Now(),
		ExpiresAt:    expiresAt,
		LastActivity: time.Now(),
		IpAddress:    ipAddress,
		UserAgent:    r.UserAgent(),
	}

	err = s.db.StoreSession(r.Context(), sess)
	if err != nil {
		log.Err(err).Msg("failed to store session")
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	// 3. Create session cookie
	s.setSessionCookie(w, sessionID, expiresAt)

	// Redirect to a protected page or home page
	http.Redirect(w, r, "/", http.StatusFound)
}

func randString(length int) string {
	// Calculate bytes needed for exact base64 length
	// base64 produces 4 chars per 3 bytes, so we need (length * 3) / 4 bytes
	byteLength := (length * 3) / 4
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		log.Fatal().Err(err).Msg("failed to generate random string")
	}
	encoded := base64.RawURLEncoding.EncodeToString(b)
	if len(encoded) > length {
		return encoded[:length]
	}
	return encoded
}

func generateSessionID() string {
	return randString(128)
}

func getIssSubHash(iss string, sub string) string {
	hash := sha256.Sum256(append([]byte(iss), []byte(sub)...))
	return base64.URLEncoding.EncodeToString(hash[:])
}
