package web

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/shopspring/decimal"
)

type mockPaymentPayload struct {
	OrderID int64           `json:"order_id"`
	Amount  decimal.Decimal `json:"amount"`
	Success bool            `json:"success"`
}

func decodeMockPaymentToken(token string) (mockPaymentPayload, error) {
	raw, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return mockPaymentPayload{}, err
	}
	var payload mockPaymentPayload
	if err = json.Unmarshal(raw, &payload); err != nil {
		return mockPaymentPayload{}, err
	}
	return payload, nil
}

func encodeMockPaymentToken(payload mockPaymentPayload) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(raw), nil
}

func (wh *WebHandler) MockBankPaymentPage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	payload, err := decodeMockPaymentToken(token)
	if token == "" || err != nil {
		wh.render(w, "payment", map[string]any{
			"User":  wh.userFromCookie(r),
			"Error": "Invalid payment token",
		})
		return
	}

	wh.render(w, "payment", map[string]any{
		"User":    wh.userFromCookie(r),
		"Token":   token,
		"OrderID": payload.OrderID,
		"Amount":  payload.Amount,
	})
}

func (wh *WebHandler) MockBankPaymentSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	payload, err := decodeMockPaymentToken(token)
	if err != nil {
		http.Redirect(w, r, "/orders?payment=failed&payment_error="+url.QueryEscape("Invalid payment token"), http.StatusSeeOther)
		return
	}

	if r.FormValue("result") == "declined" {
		payload.Success = false
		token, err = encodeMockPaymentToken(payload)
		if err != nil {
			http.Redirect(w, r, fmt.Sprintf("/orders/%d?payment=failed&payment_error=%s", payload.OrderID, url.QueryEscape(err.Error())), http.StatusSeeOther)
			return
		}
	}

	redirectURL := fmt.Sprintf("/orders/%d", payload.OrderID)
	if err = wh.paymentService.ProcessOrderPayment(r.Context(), token); err != nil {
		http.Redirect(w, r, redirectURL+"?payment=failed&payment_error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, redirectURL+"?payment=success", http.StatusSeeOther)
}
