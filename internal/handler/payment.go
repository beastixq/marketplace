package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
}

func NewPaymentHandler(ps *service.PaymentService) PaymentHandler {
	return PaymentHandler{paymentService: ps}
}

// POST /api/v1/orders/:id/payment-link
func (ph PaymentHandler) GetPaymentLink(w http.ResponseWriter, r *http.Request) {
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

	paymentURL, expiresAt, err := ph.paymentService.GetOrderPaymentURL(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, PaymentLinkResponse{
		OrderID:    id,
		PaymentURL: paymentURL,
		ExpiresAt:  expiresAt,
	})
}

type MockBankCallbackRequest struct {
	Token string `json:"token"`
}

// POST /api/v1/payments/callback/mock-bank
func (ph PaymentHandler) MockBankCallback(w http.ResponseWriter, r *http.Request) {
	var req MockBankCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	if err := ph.paymentService.ProcessOrderPayment(r.Context(), req.Token); err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "payment processed"})
}
