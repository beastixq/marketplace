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
	expirer := mock_service.NewMockOrderExpirer(ctrl)

	paymentTTL := 15 * time.Minute
	now := someTime

	expectedDeadline := now.Add(-paymentTTL)
	expirer.EXPECT().
		ExpireOrders(gomock.Any(), expectedDeadline).
		Return(nil).
		MinTimes(1)

	worker := service.NewOrderExpirationWorker(
		expirer,
		50*time.Millisecond,
		paymentTTL,
		slog.Default(),
	)
	worker.Clock = fakeClock{now: now}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	worker.Run(ctx)
}

func TestOrderExpirationWorker_RunLogsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	expirer := mock_service.NewMockOrderExpirer(ctrl)

	paymentTTL := 15 * time.Minute
	now := someTime

	expectedDeadline := now.Add(-paymentTTL)
	expirer.EXPECT().
		ExpireOrders(gomock.Any(), expectedDeadline).
		Return(errors.New("db connection lost")).
		MinTimes(1)

	worker := service.NewOrderExpirationWorker(
		expirer,
		50*time.Millisecond,
		paymentTTL,
		slog.Default(),
	)
	worker.Clock = fakeClock{now: now}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	worker.Run(ctx)
}
