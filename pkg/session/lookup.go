package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/htwr-aachen/backend/pkg/schema"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

type NoSessionError struct{}

func (e NoSessionError) Error() string {
	return "session not found or expired"
}

func (db *DB) GetUserById(ctx context.Context, id string) (*schema.User, error) {
	if err := validation.Validate.Var(id, "uuid"); err != nil {
		return nil, fmt.Errorf("invalid user id format: %w", err)
	}

	userObj, found := db.cache.Get(id)
	if user, ok := userObj.(schema.User); found && ok {
		return &user, nil
	}

	query := `
	SELECT u.id, u.iss_sub_hash, u.username, u.name, u.role, u.email, u.avatar_url, u.identity_provider, u.created_at, u.updated_at
	FROM users u
	WHERE u.id = $1
	`

	var user schema.User
	err := db.sql.QueryRow(ctx, query, id).Scan(
		&user.Id,
		&user.IssSubHash,
		&user.Username,
		&user.Name,
		&user.Role,
		&user.Email,
		&user.AvatarURL,
		&user.IdentityProvider,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, schema.NoUserError{}
		}
		return nil, fmt.Errorf("failed to lookup user by id: %w", err)
	}

	db.cache.Set(user.Id, user, cache.DefaultExpiration)

	return &user, nil
}

func (db *DB) LookupSession(ctx context.Context, sessionId string) (*schema.User, error) {
	if err := validation.Validate.Var(sessionId, "len=128"); err != nil {
		return nil, fmt.Errorf("invalid session_id format: %w", err)
	}

	sessionObj, found := db.cache.Get(sessionId)
	if session, ok := sessionObj.(schema.Session); found && ok {
		return db.GetUserById(ctx, session.UserId)
	}

	query := `
	SELECT u.id, u.iss_sub_hash, u.username, u.name, u.role, u.email, u.avatar_url, u.identity_provider, u.created_at, u.updated_at
	FROM users u
	INNER JOIN sessions s ON u.id = s.user_id
	WHERE s.session_id = $1 AND s.expires_at > NOW()
	`

	var user schema.User
	err := db.sql.QueryRow(ctx, query, sessionId).Scan(
		&user.Id,
		&user.IssSubHash,
		&user.Username,
		&user.Name,
		&user.Role,
		&user.Email,
		&user.AvatarURL,
		&user.IdentityProvider,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, schema.NoSessionError{}
		}
		return nil, fmt.Errorf("failed to lookup user by session: %w", err)
	}

	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err = db.sql.Exec(updateCtx,
			"UPDATE sessions SET last_activity = NOW() WHERE session_id = $1",
			sessionId)
		if err != nil {
			log.Err(err).Str("session_id", sessionId).Msg("updating last_activity on session")
		}
	}()

	db.cache.Set(user.Id, user, cache.DefaultExpiration)

	return &user, nil
}

func (db *DB) GetValidSessionById(ctx context.Context, sessionId string) (*schema.Session, error) {
	if err := validation.Validate.Var(sessionId, "len=128"); err != nil {
		return nil, fmt.Errorf("invalid session_id format: %w", err)
	}

	query := `
	SELECT session_id, user_id, refresh_token, id_token, access_token,
	       token_expires_at, created_at, expires_at, last_activity,
	       ip_address, user_agent
	FROM sessions
	WHERE session_id = $1 AND expires_at > NOW()
	`

	var session schema.Session
	err := db.sql.QueryRow(ctx, query, sessionId).Scan(
		&session.SessionId,
		&session.UserId,
		&session.RefreshToken,
		&session.IdToken,
		&session.AccessToken,
		&session.TokenExpiresAt,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LastActivity,
		&session.IpAddress,
		&session.UserAgent,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NoSessionError{}
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	go func() {
		_, err = db.sql.Exec(ctx,
			"UPDATE sessions SET last_activity = NOW() WHERE session_id = $1",
			sessionId)
		if err != nil {
			log.Err(err).Str("session_id", sessionId).Msg("updating last_activity on session")
		}
	}()

	return &session, nil
}
