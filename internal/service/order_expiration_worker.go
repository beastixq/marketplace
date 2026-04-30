package service

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_order_expirer.go github.com/beastixq/marketplace/internal/service OrderExpirer
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

func (w *OrderExpirationWorker) logExpireError(err error) {
	type joined interface{ Unwrap() []error }
	if u, ok := err.(joined); ok {
		for _, e := range u.Unwrap() {
			w.logSingleExpireError(e)
		}
		return
	}
	w.logSingleExpireError(err)
}

func (w *OrderExpirationWorker) logSingleExpireError(err error) {
	var ee *ExpireOrderError
	if errors.As(err, &ee) {
		w.logger.Error("expire order failed",
			"order_id", ee.OrderID,
			"stage", ee.Stage,
			"product_id", ee.ProductID,
			"quantity", ee.Quantity,
			"reserved_quantity", ee.ReservedQuantity,
			"err", ee.Cause,
		)
		return
	}
	w.logger.Error("failed to expire orders", "err", err)
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
				w.logExpireError(err)
			}
		}
	}
}
