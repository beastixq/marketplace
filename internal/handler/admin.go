package handler

import (
	"encoding/json"
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	user, err := ah.userService.GetUserByID(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userDTO(user))
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

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

	user, err := ah.userService.UpdateUser(r.Context(), actor, id, model.UserUpdate{
		Email:    req.Email,
		FullName: req.FullName,
		Phone:    req.Phone,
		Role:     req.Role,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userDTO(user))
}

// DELETE /api/v1/admin/users/:id
func (ah AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := ah.userService.DeleteUserByID(r.Context(), actor, id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/admin/sellers/:id
func (ah AdminHandler) DeleteSeller(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := ah.sellerService.DeleteSellerByID(r.Context(), actor, id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
