package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/beastixq/marketplace/internal/middleware"
	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
)

type AddressHandler struct {
	addressService service.AddressService
}

func NewAddressHandler(addressSvc service.AddressService) AddressHandler {
	return AddressHandler{addressService: addressSvc}
}

// GET /api/v1/addresses
func (ah AddressHandler) GetAddresses(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	addrs, err := ah.addressService.GetAddressesByUserID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	result := make([]Address, len(addrs))
	for i := range addrs {
		result[i] = addressFromService(addrs[i])
	}
	writeJSON(w, http.StatusOK, result)
}

type CreateAddressRequest struct {
	City      string `json:"city"`
	Street    string `json:"street"`
	ZipCode   string `json:"zip_code"`
	IsDefault bool   `json:"is_default"`
}

func (cr CreateAddressRequest) Validate() error {
	if cr.City == "" {
		return ErrAddressCityRequired
	}
	if cr.Street == "" {
		return ErrAddressStreetRequired
	}
	if cr.ZipCode == "" {
		return ErrAddressZipCodeRequired
	}
	return nil
}

// POST /api/v1/addresses
func (ah AddressHandler) CreateAddress(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	var req CreateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := ah.addressService.CreateAddress(r.Context(), model.AddressCreate{
		UserID:    claims.UserID,
		City:      req.City,
		Street:    req.Street,
		ZipCode:   req.ZipCode,
		IsDefault: req.IsDefault,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

type UpdateAddressRequest struct {
	City      *string `json:"city"`
	Street    *string `json:"street"`
	ZipCode   *string `json:"zip_code"`
	IsDefault *bool   `json:"is_default"`
}

func (ur UpdateAddressRequest) Validate() error {
	if ur.City == nil && ur.Street == nil && ur.ZipCode == nil && ur.IsDefault == nil {
		return ErrUpdateAddressAllNil
	}
	if ur.City != nil && *ur.City == "" {
		return ErrAddressCityRequired
	}
	if ur.Street != nil && *ur.Street == "" {
		return ErrAddressStreetRequired
	}
	if ur.ZipCode != nil && *ur.ZipCode == "" {
		return ErrAddressZipCodeRequired
	}
	return nil
}

// PATCH /api/v1/addresses/:id
func (ah AddressHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	var req UpdateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	addr, err := ah.addressService.UpdateAddress(r.Context(), claims.UserID, id, model.AddressUpdate{
		City:      req.City,
		Street:    req.Street,
		ZipCode:   req.ZipCode,
		IsDefault: req.IsDefault,
	})
	if err != nil {
		if errors.Is(err, service.ErrAddressNotFound) {
			writeError(w, http.StatusNotFound, service.ErrAddressNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourAddress) {
			writeError(w, http.StatusForbidden, service.ErrNotYourAddress.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, addressFromService(addr))
}

// DELETE /api/v1/addresses/:id
func (ah AddressHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := ah.addressService.DeleteAddressByID(r.Context(), claims.UserID, id); err != nil {
		if errors.Is(err, service.ErrAddressNotFound) {
			writeError(w, http.StatusNotFound, service.ErrAddressNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourAddress) {
			writeError(w, http.StatusForbidden, service.ErrNotYourAddress.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
