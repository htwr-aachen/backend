package schema

import (
	"fmt"
	"time"
)

type User struct {
	Id               string    `json:"id" validate:"required,uuid"`
	IssSubHash       string    `json:"iss_sub_hash" validate:"base64"`
	Username         string    `json:"username" validate:"required,lt=255"`
	Name             string    `json:"name"`
	Role             UserRole  `json:"role"`
	Email            string    `json:"email" validate:"required,email"`
	AvatarURL        string    `json:"avatar_url" validate:"url"`
	IdentityProvider string    `json:"identity_provider"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type NoUserError struct{}

func (err NoUserError) Error() string {
	return "no such user found"
}

type Unauthenticated struct{}

func (e Unauthenticated) Error() string {
	return "user not logged in"
}

type Unauthorized struct {
	Role         string
	RequiredRole string
}

func (e Unauthorized) Error() string {
	return fmt.Sprintf("User of role %s cannot access resource of required role %s", e.Role, e.RequiredRole)
}
