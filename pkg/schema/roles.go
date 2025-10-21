package schema

import (
	"database/sql/driver"
	"fmt"
)

const (
	ROLE_USER      UserRole = 0
	ROLE_READONLY  UserRole = 10
	ROLE_EDITOR    UserRole = 20
	ROLE_ADMIN     UserRole = 30
	HIGHSTEST_ROLE          = 30
)

const (
	ROLE_NAME_USER     = "user"
	ROLE_NAME_READONLY = "readonly"
	ROLE_NAME_EDITOR   = "editor"
	ROLE_NAME_ADMIN    = "admin"
)

type UserRole byte

// Implement driver.Valuer to encode UserRole for database storage
func (r UserRole) Value() (driver.Value, error) {

	// Convert the role to its string representation
	switch r {
	case ROLE_ADMIN:
		return ROLE_NAME_ADMIN, nil
	case ROLE_EDITOR:
		return ROLE_NAME_EDITOR, nil
	case ROLE_READONLY:
		return ROLE_NAME_READONLY, nil
	case ROLE_USER:
		return ROLE_NAME_USER, nil
	default:
		return ROLE_NAME_USER, nil // Default to user
	}
}

// Implement sql.Scanner to decode from database
func (r *UserRole) Scan(value interface{}) error {
	if value == nil {
		*r = ROLE_USER
		return nil
	}

	str, ok := value.(string)
	if !ok {
		bytes, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("failed to scan UserRole: expected string or []byte, got %T", value)
		}
		str = string(bytes)
	}

	switch str {
	case ROLE_NAME_ADMIN:
		*r = ROLE_ADMIN
	case ROLE_NAME_EDITOR:
		*r = ROLE_EDITOR
	case ROLE_NAME_READONLY:
		*r = ROLE_READONLY
	case ROLE_NAME_USER:
		*r = ROLE_USER
	default:
		*r = ROLE_USER
	}

	return nil
}
