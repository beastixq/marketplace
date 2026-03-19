package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	userService   service.UserService
	sellerService service.SellerService
}

func NewAdminHandler(userSvc service.UserService, sellerSvc service.SellerService) AdminHandler {
	return AdminHandler{userService: userSvc, sellerService: sellerSvc}
}

// GET /api/v1/admin/users
func (ah AdminHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// TODO: add pagination support when UserService gets a GetAllUsers method
	writeError(w, http.StatusNotImplemented, "not implemented")
}

// GET /api/v1/admin/users/:id
func (ah AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	user, err := ah.userService.GetUserByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, service.ErrUserNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, userFromService(user))
}

type AdminUpdateUserRequest struct {
	Email    *string         `json:"email"`
	FullName *string         `json:"full_name"`
	Phone    *string         `json:"phone"`
	Role     *model.UserRole `json:"role"`
}

func (ur AdminUpdateUserRequest) Validate() error {
	if ur.Email == nil && ur.FullName == nil && ur.Phone == nil && ur.Role == nil {
		return ErrUpdateProfileAllNil
	}
	return nil
}

// PATCH /api/v1/admin/users/:id
func (ah AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	var req AdminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := ah.userService.UpdateUser(r.Context(), id, model.UserUpdate{
		Email:    req.Email,
		FullName: req.FullName,
		Phone:    req.Phone,
		Role:     req.Role,
	})
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, service.ErrUserNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, userFromService(user))
}

// DELETE /api/v1/admin/users/:id
func (ah AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := ah.userService.DeleteUserByID(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, service.ErrUserNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/admin/sellers/:id
func (ah AdminHandler) DeleteSeller(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	// admin bypasses ownership check — pass 0 as userID, service needs adjustment
	// TODO: add AdminDeleteSeller to SellerService that skips ownership check
	if err := ah.sellerService.DeleteSellerByID(r.Context(), 0, id); err != nil {
		if errors.Is(err, service.ErrSellerNotFound) {
			writeError(w, http.StatusNotFound, service.ErrSellerNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
