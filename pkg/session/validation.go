package session

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/htwr-aachen/backend/internal/validation"
)

func validateId(id string) error {
	return validation.Validate.Var(id, "uuid")
}

func formatValidationError(err error) error {
	if validationErrors, ok := errors.AsType[validator.ValidationErrors](err); ok {
		var errMessages []string

		for _, e := range validationErrors {
			// Get the full namespace (e.g., "SessionConfig.Providers[google].ClientId")
			namespace := e.Namespace()
			// Remove the root struct name for cleaner output
			field := strings.TrimPrefix(namespace, "SessionConfig.")
			tag := e.Tag()
			param := e.Param()

			var msg string
			switch tag {
			case "required":
				msg = fmt.Sprintf("%s is required", field)
			case "min":
				if e.Type().String() == "time.Duration" {
					msg = fmt.Sprintf("%s must be at least %s", field, param)
				} else {
					msg = fmt.Sprintf("%s must have minimum length/value of %s", field, param)
				}
			case "url":
				msg = fmt.Sprintf("%s must be a valid URL", field)
			case "dive":
				// Skip dive errors, they're usually redundant
				continue
			default:
				msg = fmt.Sprintf("%s failed validation '%s'", field, tag)
			}

			errMessages = append(errMessages, msg)
		}

		if len(errMessages) == 0 {
			return fmt.Errorf("validation failed")
		}

		if len(errMessages) == 1 {
			return fmt.Errorf("validation failed: %s", errMessages[0])
		}

		return fmt.Errorf("validation failed:\n  - %s", strings.Join(errMessages, "\n  - "))
	}
	return fmt.Errorf("validation error: %w", err)
}
