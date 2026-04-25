package service

import (
	"context"
	"errors"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/port"
)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type PaymentService struct {
	orderRepo  OrderRepo
	gateway    port.PaymentGateway
	paymentTTL time.Duration
	Clock      Clock
}

func NewPaymentService(
	orderRepo OrderRepo,
	gateway port.PaymentGateway,
	paymentTTL time.Duration,
) *PaymentService {
	return &PaymentService{
		orderRepo:  orderRepo,
		gateway:    gateway,
		paymentTTL: paymentTTL,
		Clock:      realClock{},
	}
}

func (ps PaymentService) GetOrderPaymentURL(ctx context.Context, orderID, userID int64) (string, time.Time, error) {
	order, err := ps.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return "", time.Time{}, err
	}

	if order.UserID != userID {
		return "", time.Time{}, ErrNotYourOrder
	}

	if order.Status != m.StatusPending {
		return "", time.Time{}, ErrOrderStatusInvalid
	}

	expiresAt := order.CreatedAt.Add(ps.paymentTTL)
	if !ps.Clock.Now().Before(expiresAt) {
		return "", time.Time{}, ErrPaymentExpired
	}

	paymentURL, err := ps.gateway.GetPaymentURL(ctx, port.PaymentPayload{
		OrderID: order.ID,
		Amount:  order.TotalAmount,
	})
	if err != nil {
		return "", time.Time{}, err
	}

	return paymentURL, expiresAt, nil
}

func (ps PaymentService) ProcessOrderPayment(ctx context.Context, token string) error {
	result, err := ps.gateway.ProcessPayment(ctx, token)
	if err != nil {
		return err
	}

	order, err := ps.orderRepo.GetOrderByID(ctx, result.OrderID)
	if err != nil {
		return err
	}

	if order.Status == m.StatusPaid {
		return nil
	}
	if order.Status == m.StatusCancelled {
		return nil
	}

	if order.Status != m.StatusPending {
		return ErrOrderStatusInvalid
	}

	expiresAt := order.CreatedAt.Add(ps.paymentTTL)
	if !ps.Clock.Now().Before(expiresAt) {
		return ErrPaymentExpired
	}

	if !result.Success {
		return ErrPaymentDeclined
	}

	if order.TotalAmount.Cmp(result.Amount) != 0 {
		return ErrInvalidPaymentAmount
	}

	if err = ps.orderRepo.UpdateOrderStatus(ctx, order.ID, []m.OrderStatus{m.StatusPending}, m.StatusPaid); err != nil {
		if errors.Is(err, ErrNotFound) {
			current, getErr := ps.orderRepo.GetOrderByID(ctx, order.ID)
			if getErr != nil {
				return getErr
			}
			if current.Status == m.StatusPaid || current.Status == m.StatusCancelled {
				return nil
			}
			return ErrOrderStatusInvalid
		}
		return err
	}

	return nil
}
