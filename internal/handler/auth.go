package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/beastixq/marketplace/internal/middleware"
	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
)

const minPassLen = 6

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authSvc service.AuthService) AuthHandler {
	return AuthHandler{authService: authSvc}
}

type RegisterRequest struct {
	Password string         `json:"password"`
	Email    string         `json:"email"`
	FullName string         `json:"full_name"`
	Phone    *string        `json:"phone"`
	Role     model.UserRole `json:"role"`
}

func (rr RegisterRequest) Validate() error {
	if err := validateFullName(rr.FullName); err != nil {
		return err
	}
	if err := validatePassword(rr.Password); err != nil {
		return err
	}
	if err := validateEmail(rr.Email); err != nil {
		return err
	}
	if rr.Phone != nil {
		if err := validatePhone(*rr.Phone); err != nil {
			return err
		}
	}
	if err := validateRole(rr.Role); err != nil {
		return err
	}
	if rr.Role == model.RoleAdmin || rr.Role == model.RoleAnalyst {
		return ErrRoleUnavailable
	}
	return nil
}

func (ah AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := ah.authService.Register(r.Context(), model.UserCreate{
		Email:    req.Email,
		FullName: req.FullName,
		Password: req.Password,
		Phone:    req.Phone,
		Role:     req.Role,
	})
	if err != nil {
		if errors.Is(err, service.ErrAccountWithEmailAlreadyExists) {
			writeError(w, http.StatusConflict, service.ErrAccountWithEmailAlreadyExists.Error())
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

type LoginRequest struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

func (lr LoginRequest) Validate() error {
	if at := strings.Index(lr.Email, "@"); at < 1 || !strings.Contains(lr.Email[at+1:], ".") || strings.HasSuffix(lr.Email, ".") {
		return ErrInvalidEmail
	}
	return nil
}

func (ah AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err = req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	token, err := ah.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrWrongPassword) {
			writeError(w, http.StatusUnauthorized, service.ErrWrongPassword.Error())
			return
		}
		if errors.Is(err, service.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, service.ErrUserNotFound.Error())
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (ah AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if err := ah.authService.Logout(r.Context(), claims); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
