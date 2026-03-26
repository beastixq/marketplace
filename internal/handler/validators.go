package handler

import (
	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/validators"
)

func validateEmail(email string) error {
	return validators.ValidateEmail(email)
}

func validateFullName(fullName string) error {
	return validators.ValidateFullName(fullName)
}

func validatePhone(phone string) error {
	return validators.ValidatePhone(phone)
}

func validateRole(role model.UserRole) error {
	return validators.ValidateRole(role)
}

func validatePassword(password string) error {
	return validators.ValidatePassword(password)
}
