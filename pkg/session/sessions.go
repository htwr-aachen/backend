package session

import (
	"context"
	"fmt"

	"github.com/htwr-aachen/backend/pkg/schema"
)

func (db *DB) StoreSession(ctx context.Context, session *schema.Session) error {
	query := `
	INSERT INTO sessions (
		session_id, user_id, refresh_token, id_token, access_token,
		token_expires_at, created_at, expires_at, last_activity,
		ip_address, user_agent
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
	) ON CONFLICT (session_id) DO UPDATE SET
		user_id = EXCLUDED.user_id,
		refresh_token = EXCLUDED.refresh_token,
		id_token = EXCLUDED.id_token,
		access_token = EXCLUDED.access_token,
		token_expires_at = EXCLUDED.token_expires_at,
		expires_at = EXCLUDED.expires_at,
		last_activity = EXCLUDED.last_activity,
		ip_address = EXCLUDED.ip_address,
		user_agent = EXCLUDED.user_agent
	`

	_, err := db.sql.Exec(ctx, query,
		session.SessionId,
		session.UserId,
		session.RefreshToken,
		session.IdToken,
		session.AccessToken,
		session.TokenExpiresAt,
		session.CreatedAt,
		session.ExpiresAt,
		session.LastActivity,
		session.IpAddress,
		session.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	return nil
}
