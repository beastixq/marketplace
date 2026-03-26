package validators

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/beastixq/marketplace/internal/model"
)

const MinPassLen = 6

var (
	ErrFullNameShouldBeTwoWords = errors.New("Full name should consist of two words")
	ErrPasswordTooShort         = fmt.Errorf("Password too short. Should be at least %d symbols", MinPassLen)
	ErrPasswordNoDigit          = errors.New("Password must contain at least one digit")
	ErrPasswordNoLetter         = errors.New("Password must contain at least one letter")
	ErrInvalidRole              = errors.New("Invalid role name")
	ErrInvalidEmail             = errors.New("Invalid email")
	ErrInvalidPhone             = errors.New("Invalid phone number")
)

func ValidateEmail(email string) error {
	if at := strings.Index(email, "@"); at < 1 || !strings.Contains(email[at+1:], ".") || strings.HasSuffix(email, ".") {
		return ErrInvalidEmail
	}
	return nil
}

func ValidateFullName(fullName string) error {
	if words := strings.Split(fullName, " "); len(words) != 2 {
		return ErrFullNameShouldBeTwoWords
	}
	return nil
}

func ValidatePhone(phone string) error {
	phone = strings.TrimPrefix(phone, "+")
	if len(phone) != 11 {
		return ErrInvalidPhone
	}
	return nil
}

func ValidateRole(role model.UserRole) error {
	if !slices.Contains([]model.UserRole{model.RoleAdmin, model.RoleAnalyst, model.RoleBuyer, model.RoleSeller}, role) {
		return ErrInvalidRole
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < MinPassLen {
		return ErrPasswordTooShort
	}
	var hasDigit, hasLetter bool
	for _, ch := range password {
		if unicode.IsDigit(ch) {
			hasDigit = true
		}
		if unicode.IsLetter(ch) {
			hasLetter = true
		}
	}
	if !hasDigit {
		return ErrPasswordNoDigit
	}
	if !hasLetter {
		return ErrPasswordNoLetter
	}
	return nil
}
