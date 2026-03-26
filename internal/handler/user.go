package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/beastixq/marketplace/internal/middleware"
	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userSvc service.UserService) UserHandler {
	return UserHandler{userService: userSvc}
}

type UserProfile struct {
	Email     string         `json:"email"`
	FullName  string         `json:"full_name"`
	Phone     *string        `json:"phone"`
	Role      model.UserRole `json:"role"`
	CreatedAt time.Time      `json:"created_at"`
}

func (uh UserHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	tokenClaims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	user, err := uh.userService.GetUserByID(r.Context(), tokenClaims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, UserProfile{
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	})
}

type UpdateProfileRequest struct {
	Email    *string `json:"email"`
	FullName *string `json:"full_name"`
	Phone    *string `json:"phone"`
}

func (ur UpdateProfileRequest) Validate() error {
	if ur.Email == nil && ur.FullName == nil && ur.Phone == nil {
		return ErrUpdateProfileAllNil
	}
	if ur.Email != nil {
		if err := validateEmail(*ur.Email); err != nil {
			return err
		}
	}
	if ur.FullName != nil {
		if err := validateFullName(*ur.FullName); err != nil {
			return err
		}
	}
	if ur.Phone != nil {
		if err := validatePhone(*ur.Phone); err != nil {
			return err
		}
	}
	return nil
}

func (uh UserHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	tokenClaims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := uh.userService.UpdateUser(r.Context(), tokenClaims.UserID, model.UserUpdate{
		FullName: req.FullName,
		Email:    req.Email,
		Phone:    req.Phone,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, userDTO(user))
}

func (uh UserHandler) DeleteMyAccount(w http.ResponseWriter, r *http.Request) {
	tokenClaims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	if err := uh.userService.DeleteUserByID(r.Context(), tokenClaims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (cr ChangePasswordRequest) Validate() error {
	return validatePassword(cr.NewPassword)
}

func (uh UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	tokenClaims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := uh.userService.ChangePasswordUser(r.Context(), tokenClaims.UserID, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrWrongPassword) {
			writeError(w, http.StatusUnauthorized, service.ErrWrongPassword.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}
