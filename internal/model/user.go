package model

import (
	"time"
)

type UserRole string

const (
	RoleAdmin   = "admin"
	RoleAnalyst = "analyst"
	RoleBuyer   = "buyer"
	RoleSeller  = "seller"
)

type User struct {
	ID        int64
	Email     string
	FullName  string
	Phone     *string
	Role      UserRole
	CreatedAt time.Time
	DeletedAt *time.Time
}

type UserCreate struct {
	PasswordHash string
	Email        string
	FullName     string
	Phone        *string
	Role         UserRole
}
type UserUpdate struct {
	Email    *string
	FullName *string
	Phone    *string
	Role     *UserRole
}
