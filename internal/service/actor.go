package service

import m "github.com/beastixq/marketplace/internal/model"

type Actor struct {
	UserID int64
	Role   m.UserRole
}

func (a Actor) IsAdmin() bool {
	return a.Role == m.RoleAdmin
}

func (a Actor) HasRole(roles ...m.UserRole) bool {
	for _, role := range roles {
		if a.Role == role {
			return true
		}
	}
	return false
}
