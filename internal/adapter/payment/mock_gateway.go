package payment

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/beastixq/marketplace/internal/port"
	"github.com/shopspring/decimal"
)

type MockBankGateway struct {
	baseURL string
}

func NewMockBankGateway(baseURL string) *MockBankGateway {
	return &MockBankGateway{baseURL: baseURL}
}

type tokenPayload struct {
	OrderID int64           `json:"order_id"`
	Amount  decimal.Decimal `json:"amount"`
	Success bool            `json:"success"`
}

func (g *MockBankGateway) GetPaymentURL(ctx context.Context, payload port.PaymentPayload) (string, error) {
	tp := tokenPayload{
		OrderID: payload.OrderID,
		Amount:  payload.Amount,
		Success: true,
	}

	raw, err := json.Marshal(tp)
	if err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(raw)
	return fmt.Sprintf("%s/mock_bank/payment?token=%s", g.baseURL, token), nil
}

func (g *MockBankGateway) ProcessPayment(ctx context.Context, token string) (port.PaymentResult, error) {
	raw, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return port.PaymentResult{}, err
	}

	var tp tokenPayload
	if err := json.Unmarshal(raw, &tp); err != nil {
		return port.PaymentResult{}, err
	}

	return port.PaymentResult{
		OrderID:    tp.OrderID,
		Amount:     tp.Amount,
		Success:    tp.Success,
		ExternalID: "mock-bank",
	}, nil
}
