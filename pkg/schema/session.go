package schema

import (
	"fmt"
	"time"
)

type Session struct {
	SessionId      string
	UserId         string
	RefreshToken   string
	IdToken        string
	AccessToken    string
	TokenExpiresAt time.Time
	CreatedAt      time.Time
	ExpiresAt      time.Time
	LastActivity   time.Time
	IpAddress      string
	UserAgent      string
}

type NoSessionError struct {
	sessionId string
}

func (err NoSessionError) Error() string {
	return fmt.Sprintf("no such session found with id %s", err.sessionId)
}
