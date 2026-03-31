package service_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	"github.com/beastixq/marketplace/internal/service"
	"go.uber.org/mock/gomock"
)

func TestOrderExpirationWorker_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	orderMock := mock_service.NewMockOrderRepo(ctrl)

	paymentTTL := 15 * time.Minute
	now := someTime

	// Worker calls CancelExpiredPendingOrders with deadline = now - paymentTTL
	expectedDeadline := now.Add(-paymentTTL)
	orderMock.EXPECT().
		CancelExpiredPendingOrders(gomock.Any(), expectedDeadline).
		Return(nil).
		MinTimes(1)

	worker := service.NewOrderExpirationWorker(
		orderMock,
		50*time.Millisecond, // short interval for test
		paymentTTL,
		slog.Default(),
	)
	worker.Clock = fakeClock{now: now}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	worker.Run(ctx)
	// Run returns when ctx is done — if we get here, the worker stopped cleanly
}

func TestOrderExpirationWorker_RunLogsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	orderMock := mock_service.NewMockOrderRepo(ctrl)

	paymentTTL := 15 * time.Minute
	now := someTime

	expectedDeadline := now.Add(-paymentTTL)
	orderMock.EXPECT().
		CancelExpiredPendingOrders(gomock.Any(), expectedDeadline).
		Return(errors.New("db connection lost")).
		MinTimes(1)

	worker := service.NewOrderExpirationWorker(
		orderMock,
		50*time.Millisecond,
		paymentTTL,
		slog.Default(),
	)
	worker.Clock = fakeClock{now: now}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Should not panic on error — just logs it
	worker.Run(ctx)
}
