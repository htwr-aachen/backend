package session

import (
	"testing"

	"github.com/htwr-aachen/backend/pkg/schema"
)

func TestSessionSubsystem_GetRoleFromGroups(t *testing.T) {
	tests := []struct {
		name    string
		roleMap map[string]string
		groups  []string
		want    schema.UserRole
	}{
		{
			name: "no groups returns default user role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{},
			want:   schema.ROLE_USER,
		},
		{
			name: "nil groups returns default user role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: nil,
			want:   schema.ROLE_USER,
		},
		{
			name: "admin group returns admin role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"admin-group"},
			want:   schema.ROLE_ADMIN,
		},
		{
			name: "editor group returns editor role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"editor-group"},
			want:   schema.ROLE_EDITOR,
		},
		{
			name: "readonly group returns readonly role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"readonly-group"},
			want:   schema.ROLE_READONLY,
		},
		{
			name: "admin and editor groups returns admin role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"admin-group", "editor-group"},
			want:   schema.ROLE_ADMIN,
		},
		{
			name: "editor and admin groups returns admin role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"editor-group", "admin-group"},
			want:   schema.ROLE_ADMIN,
		},
		{
			name: "editor and readonly groups returns editor role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"editor-group", "readonly-group"},
			want:   schema.ROLE_EDITOR,
		},
		{
			name: "readonly and editor groups returns editor role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"readonly-group", "editor-group"},
			want:   schema.ROLE_EDITOR,
		},
		{
			name: "all role groups returns admin role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"readonly-group", "editor-group", "admin-group"},
			want:   schema.ROLE_ADMIN,
		},
		{
			name: "unrecognized groups return default user role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"unknown-group", "another-group"},
			want:   schema.ROLE_USER,
		},
		{
			name: "mix of recognized and unrecognized groups",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"unknown-group", "editor-group", "another-group"},
			want:   schema.ROLE_EDITOR,
		},
		{
			name: "stops at highest role",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"admin-group", "more-groups", "even-more"},
			want:   schema.ROLE_ADMIN,
		},
		{
			name:    "empty role map",
			roleMap: map[string]string{},
			groups:  []string{"admin-group", "editor-group", "readonly-group"},
			want:    schema.ROLE_USER,
		},
		{
			name:    "nil role map",
			roleMap: nil,
			groups:  []string{"admin-group", "editor-group", "readonly-group"},
			want:    schema.ROLE_USER,
		},
		{
			name: "case sensitive group matching",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "Admin-Group",
				schema.ROLE_NAME_EDITOR:   "Editor-Group",
				schema.ROLE_NAME_READONLY: "ReadOnly-Group",
			},
			groups: []string{"admin-group", "editor-group", "readonly-group"},
			want:   schema.ROLE_USER,
		},
		{
			name: "duplicate groups",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "admin-group",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"editor-group", "editor-group", "readonly-group"},
			want:   schema.ROLE_EDITOR,
		},
		{
			name: "partial role map with admin only",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN: "admin-group",
			},
			groups: []string{"admin-group", "editor-group"},
			want:   schema.ROLE_ADMIN,
		},
		{
			name: "partial role map with editor only",
			roleMap: map[string]string{
				schema.ROLE_NAME_EDITOR: "editor-group",
			},
			groups: []string{"editor-group", "readonly-group"},
			want:   schema.ROLE_EDITOR,
		},
		{
			name: "empty string group names",
			roleMap: map[string]string{
				schema.ROLE_NAME_ADMIN:    "",
				schema.ROLE_NAME_EDITOR:   "editor-group",
				schema.ROLE_NAME_READONLY: "readonly-group",
			},
			groups: []string{"", "editor-group"},
			want:   schema.ROLE_EDITOR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SessionSubsystem{
				config: SessionConfig{
					RoleMap: tt.roleMap,
				},
			}
			got := s.GetRoleFromGroups(tt.groups)
			if got != tt.want {
				t.Errorf("GetRoleFromGroups() = %v, want %v", got, tt.want)
			}
		})
	}
}
