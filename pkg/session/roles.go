package session

import (
	"strings"

	"github.com/htwr-aachen/backend/pkg/schema"
)

func (s *SessionSubsystem) GetRoleFromGroups(groups []string) schema.UserRole {
	highestRole := schema.ROLE_USER

	for _, group := range groups {
		if highestRole >= schema.HIGHSTEST_ROLE {
			break
		}

		roleMap := strings.TrimSpace(s.config.RoleMap[schema.ROLE_NAME_ADMIN])
		if highestRole < schema.ROLE_ADMIN && roleMap != "" && group == roleMap {
			highestRole = schema.ROLE_ADMIN
		}
		roleMap = strings.TrimSpace(s.config.RoleMap[schema.ROLE_NAME_EDITOR])
		if highestRole < schema.ROLE_EDITOR && roleMap != "" && group == roleMap {
			highestRole = schema.ROLE_EDITOR
		}
		roleMap = strings.TrimSpace(s.config.RoleMap[schema.ROLE_NAME_READONLY])
		if highestRole < schema.ROLE_READONLY && roleMap != "" && group == roleMap {
			highestRole = schema.ROLE_READONLY
		}
	}

	return highestRole
}
