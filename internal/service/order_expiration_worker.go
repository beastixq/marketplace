package service

import (
	"context"
	"log/slog"
	"time"
)

type OrderExpirer interface {
	ExpireOrders(ctx context.Context, deadline time.Time) error
}

type OrderExpirationWorker struct {
	expirer    OrderExpirer
	interval   time.Duration
	paymentTTL time.Duration
	logger     *slog.Logger
	Clock      Clock
}

func NewOrderExpirationWorker(
	expirer OrderExpirer,
	interval time.Duration,
	paymentTTL time.Duration,
	logger *slog.Logger,
) *OrderExpirationWorker {
	return &OrderExpirationWorker{
		expirer:    expirer,
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
			if err := w.expirer.ExpireOrders(ctx, deadline); err != nil {
				w.logger.Error("failed to expire orders", "err", err)
			}
		}
	}
}
