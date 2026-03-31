package port

import (
	"context"

	"github.com/shopspring/decimal"
)

type PaymentPayload struct {
	OrderID int64
	Amount  decimal.Decimal
}

type PaymentResult struct {
	OrderID       int64
	Amount        decimal.Decimal
	Success       bool
	ExternalID    string
	FailureReason string
}

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_payment_gateway.go github.com/beastixq/marketplace/internal/port PaymentGateway
type PaymentGateway interface {
	GetPaymentURL(ctx context.Context, payload PaymentPayload) (string, error)
	ProcessPayment(ctx context.Context, token string) (PaymentResult, error)
}
