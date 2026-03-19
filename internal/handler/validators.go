package handler

import (
	"slices"
	"strings"

	"github.com/beastixq/marketplace/internal/model"
)

func validateEmail(email string) error {
	if at := strings.Index(email, "@"); at < 1 || !strings.Contains(email[at+1:], ".") || strings.HasSuffix(email, ".") {
		return ErrInvalidEmail
	}
	return nil
}

func validateFullName(fullName string) error {
	if words := strings.Split(fullName, " "); len(words) != 2 {
		return ErrFullNameShouldBeTwoWords
	}
	return nil
}

func validatePhone(phone string) error {
	phone = strings.TrimPrefix(phone, "+")
	if len(phone) != 11 {
		return ErrInvalidPhone
	}
	return nil
}

func validateRole(role model.UserRole) error {
	if !slices.Contains([]model.UserRole{model.RoleAdmin, model.RoleAnalyst, model.RoleBuyer, model.RoleSeller}, role) {
		return ErrInvalidRole
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < minPassLen {
		return ErrPasswordTooShort
	}
	return nil
}
