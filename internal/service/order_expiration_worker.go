package service

import (
	"context"
	"log/slog"
	"time"
)

type OrderExpirationWorker struct {
	orderRepo  OrderRepo
	interval   time.Duration
	paymentTTL time.Duration
	logger     *slog.Logger
	Clock      Clock
}

func NewOrderExpirationWorker(
	orderRepo OrderRepo,
	interval time.Duration,
	paymentTTL time.Duration,
	logger *slog.Logger,
) *OrderExpirationWorker {
	return &OrderExpirationWorker{
		orderRepo:  orderRepo,
		interval:   interval,
		paymentTTL: paymentTTL,
		logger:     logger,
		Clock:      realClock{},
	}
}

func (w *OrderExpirationWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deadline := w.Clock.Now().Add(-w.paymentTTL)
			if err := w.orderRepo.CancelExpiredPendingOrders(ctx, deadline); err != nil {
				w.logger.Error("failed to cancel expired pending orders", "err", err)
			}
		}
	}
}
