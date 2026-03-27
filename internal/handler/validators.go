package handler

import (
	"net/http"
	"strconv"

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

func parsePagination(r *http.Request) (model.PaginationOpts, error) {
	q := r.URL.Query()
	if !q.Has("page") && !q.Has("limit") {
		return model.PaginationOpts{}, nil
	}
	if q.Has("page") && !q.Has("limit") {
		return model.PaginationOpts{}, ErrNoLimitInPaginationOptions
	}
	if !q.Has("page") && q.Has("limit") {
		return model.PaginationOpts{}, ErrNoPageInPaginationOptions
	}
	page, err := strconv.Atoi(q.Get("page"))
	if err != nil || page < 1 {
		return model.PaginationOpts{}, ErrInvalidPagePaginationOption
	}
	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit < 1 {
		return model.PaginationOpts{}, ErrInvalidLimitPaginationOption
	}
	return model.PaginationOpts{Page: page, Limit: limit}, nil
}
