package service

import m "github.com/beastixq/marketplace/internal/model"

type Actor struct {
	UserID int64
	Role   m.UserRole
}

func (a Actor) IsAdmin() bool {
	return a.Role == m.RoleAdmin
}
