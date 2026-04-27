package service

import (
	"context"
	"fmt"

	m "github.com/beastixq/marketplace/internal/model"
)

type BackofficeRepo interface {
	GetAdminOrders(ctx context.Context, opts m.AdminOrderListOptions) ([]m.Order, error)
	GetPlatformStats(ctx context.Context) (m.PlatformStats, error)
}

type BackofficeService struct {
	backofficeRepo BackofficeRepo
}

func NewBackofficeService(backofficeRepo BackofficeRepo) BackofficeService {
	return BackofficeService{backofficeRepo: backofficeRepo}
}

func (bs BackofficeService) GetAdminOrders(ctx context.Context, actor Actor, opts m.AdminOrderListOptions) ([]m.Order, error) {
	if !actor.IsAdmin() {
		return nil, ErrPermissionDenied
	}
	if opts.Status != nil && !isVisibleAdminOrderStatus(*opts.Status) {
		return nil, ErrOrderStatusInvalid
	}

	orders, err := bs.backofficeRepo.GetAdminOrders(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetAdminOrders, err)
	}
	return orders, nil
}

func (bs BackofficeService) GetPlatformStats(ctx context.Context, actor Actor) (m.PlatformStats, error) {
	if !actor.HasRole(m.RoleAdmin, m.RoleAnalyst) {
		return m.PlatformStats{}, ErrPermissionDenied
	}

	stats, err := bs.backofficeRepo.GetPlatformStats(ctx)
	if err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrGetPlatformStats, err)
	}
	return stats, nil
}

func isVisibleAdminOrderStatus(status m.OrderStatus) bool {
	switch status {
	case m.StatusPending, m.StatusPaid, m.StatusShipped, m.StatusDelivered, m.StatusCancelled:
		return true
	default:
		return false
	}
}
