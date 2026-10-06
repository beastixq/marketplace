package service_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	"github.com/beastixq/marketplace/internal/service"
	"go.uber.org/mock/gomock"
)

func TestOrderExpirationWorker_Run(t *testing.T) {
	paymentTTL := 15 * time.Minute
	tests := []struct {
		name     string
		err      error
		wantLogs []string
	}{
		{name: "expires orders using payment deadline"},
		{name: "logs dependency failure", err: errors.New("db connection lost"), wantLogs: []string{"failed to expire orders", "db connection lost"}},
		{
			name: "logs each structured failure",
			err: errors.Join(
				&service.ExpireOrderError{OrderID: 10, Stage: service.ExpireStageGetItems, Cause: errors.New("items failed")},
				&service.ExpireOrderError{OrderID: 20, ProductID: 30, Quantity: 2, ReservedQuantity: 1, Stage: service.ExpireStageReleaseStock, Cause: errors.New("release failed")},
			),
			wantLogs: []string{`"order_id":10`, `"order_id":20`, `"product_id":30`, `"quantity":2`, `"reserved_quantity":1`, "items failed", "release failed"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expirer := mock_service.NewMockOrderExpirer(gomock.NewController(t))
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			expirer.EXPECT().ExpireOrders(ctx, someTime.Add(-paymentTTL)).DoAndReturn(func(context.Context, time.Time) error {
				cancel()
				return tc.err
			}).MinTimes(1)
			var logs bytes.Buffer
			worker := service.NewOrderExpirationWorker(expirer, time.Millisecond, paymentTTL, slog.New(slog.NewJSONHandler(&logs, nil)))
			worker.Clock = fakeClock{now: someTime}

			worker.Run(ctx)

			for _, fragment := range tc.wantLogs {
				if !strings.Contains(logs.String(), fragment) {
					t.Fatalf("expected %q in logs, got %q", fragment, logs.String())
				}
			}
			if tc.err == nil && logs.Len() != 0 {
				t.Fatalf("unexpected error logs: %q", logs.String())
			}
		})
	}
	t.Run("already cancelled context stops without expiring", func(t *testing.T) {
		expirer := mock_service.NewMockOrderExpirer(gomock.NewController(t))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		worker := service.NewOrderExpirationWorker(expirer, time.Hour, paymentTTL, slog.Default())

		worker.Run(ctx)
	})
}
