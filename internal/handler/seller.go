package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/beastixq/marketplace/internal/middleware"
	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
)

type SellerHandler struct {
	sellerService service.SellerService
}

func NewSellerHandler(sellerSvc service.SellerService) SellerHandler {
	return SellerHandler{sellerService: sellerSvc}
}

// GET /api/v1/sellers/:id
func (sh SellerHandler) GetSellerByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}
	seller, err := sh.sellerService.GetSellerByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrSellerNotFound) {
			writeError(w, http.StatusNotFound, service.ErrSellerNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, sellerFromService(seller))
}

// POST /api/v1/sellers
func (sh SellerHandler) CreateSeller(w http.ResponseWriter, r *http.Request) {
	tokenClaims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	var req CreateSellerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := sh.sellerService.CreateSeller(r.Context(), model.SellerCreate{
		UserID:      tokenClaims.UserID,
		CompanyName: req.CompanyName,
		Description: req.Description,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// PATCH /api/v1/sellers/:id
func (sh SellerHandler) UpdateSeller(w http.ResponseWriter, r *http.Request) {
	tokenClaims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	sellerID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	var req UpdateSellerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	seller, err := sh.sellerService.UpdateSeller(r.Context(), tokenClaims.UserID, sellerID, model.SellerUpdate{
		CompanyName: req.CompanyName,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, service.ErrSellerNotFound) {
			writeError(w, http.StatusNotFound, service.ErrSellerNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourSeller) {
			writeError(w, http.StatusForbidden, service.ErrNotYourSeller.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, sellerFromService(seller))
}

// DELETE /api/v1/sellers/:id
func (sh SellerHandler) DeleteSeller(w http.ResponseWriter, r *http.Request) {
	tokenClaims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	sellerID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := sh.sellerService.DeleteSellerByID(r.Context(), tokenClaims.UserID, sellerID); err != nil {
		if errors.Is(err, service.ErrSellerNotFound) {
			writeError(w, http.StatusNotFound, service.ErrSellerNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourSeller) {
			writeError(w, http.StatusForbidden, service.ErrNotYourSeller.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/sellers/:id/stats
func (sh SellerHandler) GetSellerStats(w http.ResponseWriter, r *http.Request) {
	tokenClaims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	sellerID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	dateFromStr := r.URL.Query().Get("date_from")
	if dateFromStr == "" {
		writeError(w, http.StatusBadRequest, ErrDateFromRequired.Error())
		return
	}
	dateFrom, err := time.Parse(time.DateOnly, dateFromStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidDateFormat.Error())
		return
	}

	dateToStr := r.URL.Query().Get("date_to")
	if dateToStr == "" {
		writeError(w, http.StatusBadRequest, ErrDateToRequired.Error())
		return
	}
	dateTo, err := time.Parse(time.DateOnly, dateToStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidDateFormat.Error())
		return
	}

	stats, err := sh.sellerService.GetSellerStats(r.Context(), tokenClaims.UserID, sellerID, dateFrom, dateTo)
	if err != nil {
		if errors.Is(err, service.ErrSellerNotFound) {
			writeError(w, http.StatusNotFound, service.ErrSellerNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourSeller) {
			writeError(w, http.StatusForbidden, service.ErrNotYourSeller.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, sellerStatsFromService(stats))
}

// --- Request DTOs ---

type CreateSellerRequest struct {
	CompanyName string  `json:"company_name"`
	Description *string `json:"description"`
}

func (cr CreateSellerRequest) Validate() error {
	if cr.CompanyName == "" {
		return ErrCompanyNameRequired
	}
	return nil
}

type UpdateSellerRequest struct {
	CompanyName *string `json:"company_name"`
	Description *string `json:"description"`
}

func (ur UpdateSellerRequest) Validate() error {
	if ur.CompanyName == nil && ur.Description == nil {
		return ErrUpdateSellerAllNil
	}
	return nil
}
