package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/htwr-aachen/backend/pkg/schema"
)

func GetTestUser() *schema.User {
	return &schema.User{

		Id:               "7caf4f5e-85f5-4dd8-aa5b-966336fe6d04",
		IssSubHash:       "fe",
		Username:         "test_user",
		Name:             "test user",
		Role:             schema.ROLE_ADMIN,
		Email:            "test_user@exampl.com",
		AvatarURL:        "https://en.wikipedia.org/wiki/File:Minion_sculpture_in_Brisbane,_2024.jpg",
		IdentityProvider: "test",
		CreatedAt:        time.Time{},
		UpdatedAt:        time.Time{},
	}
}

func (db *DB) GetUserByIssSubHash(ctx context.Context, iss string, sub string) (*schema.User, error) {
	issSubHash := getIssSubHash(iss, sub)

	query := `
	SELECT u.id, u.iss_sub_hash, u.username, u.name, u.role, u.email, u.avatar_url, u.identity_provider, u.created_at, u.updated_at
	FROM users u
	WHERE u.iss_sub_hash = $1
	`

	var user schema.User
	err := db.sql.QueryRow(ctx, query, issSubHash).Scan(
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
			return nil, schema.NoUserError{}
		}
		return nil, fmt.Errorf("failed to lookup user by iss sub: %w", err)
	}

	return &user, nil
}

func (db *DB) CreateUser(ctx context.Context, user *schema.User) (*schema.User, error) {
	user.Id = uuid.Must(uuid.NewV7()).String()

	query := `
	INSERT INTO users (
		id, iss_sub_hash, username, name, role, email, avatar_url, identity_provider, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
	) RETURNING id, iss_sub_hash, username, name, role, email, avatar_url, identity_provider, created_at, updated_at
	`

	err := db.sql.QueryRow(ctx, query,
		user.Id,
		user.IssSubHash,
		user.Username,
		user.Name,
		user.Role,
		user.Email,
		user.AvatarURL,
		user.IdentityProvider,
	).Scan(
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
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}
