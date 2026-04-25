package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/port"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"
)

type fakeClock struct {
	now time.Time
}

func (fc fakeClock) Now() time.Time { return fc.now }

func newPaymentService(ctrl *gomock.Controller, clock service.Clock) (
	*service.PaymentService,
	*mock_service.MockOrderRepo,
	*mock_service.MockPaymentGateway,
) {
	orderMock := mock_service.NewMockOrderRepo(ctrl)
	gwMock := mock_service.NewMockPaymentGateway(ctrl)
	ps := service.NewPaymentService(orderMock, gwMock, 15*time.Minute)
	ps.Clock = clock
	return ps, orderMock, gwMock
}

// ==================== GetOrderPaymentURL ====================

func TestGetOrderPaymentURL(t *testing.T) {
	ctx := context.Background()
	createdAt := someTime
	clock := fakeClock{now: createdAt.Add(5 * time.Minute)} // 5 min after creation, within TTL

	pendingOrder := m.Order{
		ID:          someOrderID,
		UserID:      someID,
		Status:      m.StatusPending,
		TotalAmount: decimal.NewFromFloat(199.98),
		CreatedAt:   createdAt,
	}

	type testCase struct {
		Description string
		OrderID     int64
		UserID      int64
		Clock       fakeClock
		MockGet     MockOrderReturn
		MockGateway *string // payment URL returned by gateway
		MockGWErr   error
		ExpectedErr error
	}

	paymentURL := "http://localhost:8080/pay?token=abc"

	tCases := []testCase{
		{
			Description: "Success",
			OrderID:     someOrderID,
			UserID:      someID,
			Clock:       clock,
			MockGet:     MockOrderReturn{Order: pendingOrder},
			MockGateway: &paymentURL,
		},
		{
			Description: "Order not found",
			OrderID:     someOrderID,
			UserID:      someID,
			Clock:       clock,
			MockGet:     MockOrderReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrNotFound,
		},
		{
			Description: "Repo error",
			OrderID:     someOrderID,
			UserID:      someID,
			Clock:       clock,
			MockGet:     MockOrderReturn{Error: errors.New("db error")},
			ExpectedErr: errors.New("db error"),
		},
		{
			Description: "Not your order",
			OrderID:     someOrderID,
			UserID:      otherUserID,
			Clock:       clock,
			MockGet:     MockOrderReturn{Order: pendingOrder},
			ExpectedErr: service.ErrNotYourOrder,
		},
		{
			Description: "Wrong status - paid",
			OrderID:     someOrderID,
			UserID:      someID,
			Clock:       clock,
			MockGet:     MockOrderReturn{Order: m.Order{ID: someOrderID, UserID: someID, Status: m.StatusPaid, CreatedAt: createdAt}},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "Payment expired",
			OrderID:     someOrderID,
			UserID:      someID,
			Clock:       fakeClock{now: createdAt.Add(20 * time.Minute)}, // past 15 min TTL
			MockGet:     MockOrderReturn{Order: pendingOrder},
			ExpectedErr: service.ErrPaymentExpired,
		},
		{
			Description: "Gateway error",
			OrderID:     someOrderID,
			UserID:      someID,
			Clock:       clock,
			MockGet:     MockOrderReturn{Order: pendingOrder},
			MockGWErr:   errors.New("gateway down"),
			ExpectedErr: errors.New("gateway down"),
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ps, orderMock, gwMock := newPaymentService(ctrl, tCase.Clock)

			orderMock.EXPECT().GetOrderByID(ctx, tCase.OrderID).Return(tCase.MockGet.Order, tCase.MockGet.Error)

			if tCase.MockGateway != nil || tCase.MockGWErr != nil {
				url := ""
				if tCase.MockGateway != nil {
					url = *tCase.MockGateway
				}
				gwMock.EXPECT().GetPaymentURL(ctx, gomock.Any()).Return(url, tCase.MockGWErr)
			}

			resultURL, expiresAt, err := ps.GetOrderPaymentURL(ctx, tCase.OrderID, tCase.UserID)
			if tCase.ExpectedErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tCase.ExpectedErr)
				}
				if errors.Is(tCase.ExpectedErr, service.ErrNotYourOrder) ||
					errors.Is(tCase.ExpectedErr, service.ErrOrderStatusInvalid) ||
					errors.Is(tCase.ExpectedErr, service.ErrPaymentExpired) ||
					errors.Is(tCase.ExpectedErr, service.ErrNotFound) {
					if !errors.Is(err, tCase.ExpectedErr) {
						t.Fatalf("expected error %v, got %v", tCase.ExpectedErr, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resultURL != paymentURL {
				t.Fatalf("expected URL %s, got %s", paymentURL, resultURL)
			}
			expectedExpires := createdAt.Add(15 * time.Minute)
			if !expiresAt.Equal(expectedExpires) {
				t.Fatalf("expected expiresAt %v, got %v", expectedExpires, expiresAt)
			}
		})
	}
}

// ==================== ProcessOrderPayment ====================

func TestProcessOrderPayment(t *testing.T) {
	ctx := context.Background()
	createdAt := someTime
	clock := fakeClock{now: createdAt.Add(5 * time.Minute)}
	amount := decimal.NewFromFloat(199.98)

	pendingOrder := m.Order{
		ID:          someOrderID,
		UserID:      someID,
		Status:      m.StatusPending,
		TotalAmount: amount,
		CreatedAt:   createdAt,
	}

	successResult := port.PaymentResult{
		OrderID: someOrderID,
		Amount:  amount,
		Success: true,
	}
	declinedResult := port.PaymentResult{
		OrderID: someOrderID,
		Amount:  amount,
		Success: false,
	}
	wrongAmountResult := port.PaymentResult{
		OrderID: someOrderID,
		Amount:  decimal.NewFromFloat(100.00),
		Success: true,
	}

	type testCase struct {
		Description      string
		Clock            fakeClock
		MockGWResult     *port.PaymentResult
		MockGWErr        error
		MockGetOrder     *MockOrderReturn
		MockUpdateStatus *error
		MockGetAfterRace *MockOrderReturn
		ExpectedErr      error
	}

	tCases := []testCase{
		{
			Description:      "Success",
			Clock:            clock,
			MockGWResult:     &successResult,
			MockGetOrder:     &MockOrderReturn{Order: pendingOrder},
			MockUpdateStatus: ptrErr(nil),
		},
		{
			Description: "Gateway decode error",
			Clock:       clock,
			MockGWErr:   errors.New("bad token"),
			ExpectedErr: errors.New("bad token"),
		},
		{
			Description:  "Order not found after gateway",
			Clock:        clock,
			MockGWResult: &successResult,
			MockGetOrder: &MockOrderReturn{Error: service.ErrNotFound},
			ExpectedErr:  service.ErrNotFound,
		},
		{
			Description:  "Already paid - idempotent",
			Clock:        clock,
			MockGWResult: &successResult,
			MockGetOrder: &MockOrderReturn{Order: m.Order{ID: someOrderID, Status: m.StatusPaid, CreatedAt: createdAt, TotalAmount: amount}},
		},
		{
			Description:  "Already cancelled - idempotent",
			Clock:        clock,
			MockGWResult: &successResult,
			MockGetOrder: &MockOrderReturn{Order: m.Order{ID: someOrderID, Status: m.StatusCancelled, CreatedAt: createdAt, TotalAmount: amount}},
		},
		{
			Description:  "Wrong status - shipped",
			Clock:        clock,
			MockGWResult: &successResult,
			MockGetOrder: &MockOrderReturn{Order: m.Order{ID: someOrderID, Status: m.StatusShipped, CreatedAt: createdAt, TotalAmount: amount}},
			ExpectedErr:  service.ErrOrderStatusInvalid,
		},
		{
			Description:  "Payment expired",
			Clock:        fakeClock{now: createdAt.Add(20 * time.Minute)},
			MockGWResult: &successResult,
			MockGetOrder: &MockOrderReturn{Order: pendingOrder},
			ExpectedErr:  service.ErrPaymentExpired,
		},
		{
			Description:  "Payment declined",
			Clock:        clock,
			MockGWResult: &declinedResult,
			MockGetOrder: &MockOrderReturn{Order: pendingOrder},
			ExpectedErr:  service.ErrPaymentDeclined,
		},
		{
			Description:  "Amount mismatch",
			Clock:        clock,
			MockGWResult: &wrongAmountResult,
			MockGetOrder: &MockOrderReturn{Order: pendingOrder},
			ExpectedErr:  service.ErrInvalidPaymentAmount,
		},
		{
			Description:      "UpdateOrderStatus fails",
			Clock:            clock,
			MockGWResult:     &successResult,
			MockGetOrder:     &MockOrderReturn{Order: pendingOrder},
			MockUpdateStatus: ptrErr(errors.New("db error")),
			ExpectedErr:      errors.New("db error"),
		},
		{
			Description:      "Concurrent callback already paid - idempotent",
			Clock:            clock,
			MockGWResult:     &successResult,
			MockGetOrder:     &MockOrderReturn{Order: pendingOrder},
			MockUpdateStatus: ptrErr(service.ErrNotFound),
			MockGetAfterRace: &MockOrderReturn{Order: m.Order{ID: someOrderID, Status: m.StatusPaid, CreatedAt: createdAt, TotalAmount: amount}},
		},
		{
			Description:      "Concurrent expiration won - idempotent",
			Clock:            clock,
			MockGWResult:     &successResult,
			MockGetOrder:     &MockOrderReturn{Order: pendingOrder},
			MockUpdateStatus: ptrErr(service.ErrNotFound),
			MockGetAfterRace: &MockOrderReturn{Order: m.Order{ID: someOrderID, Status: m.StatusCancelled, CreatedAt: createdAt, TotalAmount: amount}},
		},
		{
			Description:      "Concurrent unsupported transition",
			Clock:            clock,
			MockGWResult:     &successResult,
			MockGetOrder:     &MockOrderReturn{Order: pendingOrder},
			MockUpdateStatus: ptrErr(service.ErrNotFound),
			MockGetAfterRace: &MockOrderReturn{Order: m.Order{ID: someOrderID, Status: m.StatusShipped, CreatedAt: createdAt, TotalAmount: amount}},
			ExpectedErr:      service.ErrOrderStatusInvalid,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ps, orderMock, gwMock := newPaymentService(ctrl, tCase.Clock)

			if tCase.MockGWResult != nil {
				gwMock.EXPECT().ProcessPayment(ctx, "test-token").Return(*tCase.MockGWResult, nil)
			} else {
				gwMock.EXPECT().ProcessPayment(ctx, "test-token").Return(port.PaymentResult{}, tCase.MockGWErr)
			}

			if tCase.MockGetOrder != nil {
				orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(tCase.MockGetOrder.Order, tCase.MockGetOrder.Error)
			}

			if tCase.MockUpdateStatus != nil {
				orderMock.EXPECT().UpdateOrderStatus(ctx, someOrderID, []m.OrderStatus{m.StatusPending}, m.StatusPaid).
					Return(*tCase.MockUpdateStatus)
			}

			if tCase.MockGetAfterRace != nil {
				orderMock.EXPECT().GetOrderByID(ctx, someOrderID).
					Return(tCase.MockGetAfterRace.Order, tCase.MockGetAfterRace.Error)
			}

			err := ps.ProcessOrderPayment(ctx, "test-token")
			if tCase.ExpectedErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				// Check sentinel errors
				if errors.Is(tCase.ExpectedErr, service.ErrOrderStatusInvalid) ||
					errors.Is(tCase.ExpectedErr, service.ErrPaymentExpired) ||
					errors.Is(tCase.ExpectedErr, service.ErrPaymentDeclined) ||
					errors.Is(tCase.ExpectedErr, service.ErrInvalidPaymentAmount) ||
					errors.Is(tCase.ExpectedErr, service.ErrNotFound) {
					if !errors.Is(err, tCase.ExpectedErr) {
						t.Fatalf("expected error %v, got %v", tCase.ExpectedErr, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
