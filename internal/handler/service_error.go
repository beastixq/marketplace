package handler

import (
	"errors"
	"net/http"

	"github.com/beastixq/marketplace/internal/service"
)

type serviceErrorResponse struct {
	err    error
	status int
}

var serviceErrorResponses = []serviceErrorResponse{
	{service.ErrPermissionDenied, http.StatusForbidden},
	{service.ErrNotYourUser, http.StatusForbidden},
	{service.ErrNotYourAddress, http.StatusForbidden},
	{service.ErrNotYourSeller, http.StatusForbidden},
	{service.ErrNotYourOrder, http.StatusForbidden},
	{service.ErrNotYourReview, http.StatusForbidden},

	{service.ErrUserNotFound, http.StatusNotFound},
	{service.ErrAddressNotFound, http.StatusNotFound},
	{service.ErrSellerNotFound, http.StatusNotFound},
	{service.ErrOrderNotFound, http.StatusNotFound},
	{service.ErrOrderItemNotFound, http.StatusNotFound},
	{service.ErrCartNotFound, http.StatusNotFound},
	{service.ErrCategoryNotFound, http.StatusNotFound},
	{service.ErrProductNotFound, http.StatusNotFound},
	{service.ErrReviewNotFound, http.StatusNotFound},
	{service.ErrNotFound, http.StatusNotFound},

	{service.ErrWrongPassword, http.StatusUnauthorized},

	{service.ErrAccountWithEmailAlreadyExists, http.StatusConflict},
	{service.ErrEmailAlreadyInUse, http.StatusConflict},
	{service.ErrPhoneAlreadyExists, http.StatusConflict},
	{service.ErrOrderStatusInvalid, http.StatusConflict},
	{service.ErrProductAlreadyInCart, http.StatusConflict},
	{service.ErrInsufficientStock, http.StatusConflict},
	{service.ErrSellerNotSet, http.StatusConflict},

	{service.ErrEmptyCart, http.StatusBadRequest},
	{service.ErrQuantityTooBig, http.StatusBadRequest},
	{service.ErrNoChangesInUpdate, http.StatusBadRequest},

	{service.ErrPaymentExpired, http.StatusGone},

	{service.ErrPaymentDeclined, http.StatusUnprocessableEntity},
	{service.ErrInvalidPaymentAmount, http.StatusUnprocessableEntity},
}

func handleServiceError(w http.ResponseWriter, err error, overrides ...serviceErrorResponse) bool {
	for _, response := range overrides {
		if errors.Is(err, response.err) {
			writeError(w, response.status, response.err.Error())
			return true
		}
	}
	for _, response := range serviceErrorResponses {
		if errors.Is(err, response.err) {
			writeError(w, response.status, response.err.Error())
			return true
		}
	}
	return false
}

func writeServiceError(w http.ResponseWriter, err error, overrides ...serviceErrorResponse) {
	if handleServiceError(w, err, overrides...) {
		return
	}
	writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
}
