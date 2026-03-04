package repository

import (
	"time"

	m "github.com/beastixq/marketplace/internal/model"
)

type userRow struct {
	ID           int64
	Email        string
	PasswordHash string
	FullName     string
	Phone        *string
	Role         m.UserRole
	CreatedAt    time.Time
	DeletedAt    *time.Time
}

func (ur userRow) toModel() m.User {
	return m.User{
		ID:        ur.ID,
		Email:     ur.Email,
		FullName:  ur.FullName,
		Phone:     ur.Phone,
		Role:      ur.Role,
		CreatedAt: ur.CreatedAt,
		DeletedAt: ur.DeletedAt,
	}
}
