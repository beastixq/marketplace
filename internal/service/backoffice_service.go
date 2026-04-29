package service

import (
	"context"
	"fmt"

	m "github.com/beastixq/marketplace/internal/model"
)

type BackofficeRepo interface {
	GetAdminOrders(ctx context.Context, opts m.AdminOrderListOptions) ([]m.Order, error)
	GetPlatformStats(ctx context.Context) (m.PlatformStats, error)
	GetOrderDynamics(ctx context.Context, opts m.ReportOptions) ([]m.OrderDynamicsPoint, error)
	GetSalesByCategory(ctx context.Context, opts m.ReportOptions) ([]m.CategorySalesStats, error)
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

func (bs BackofficeService) GetOrderDynamics(ctx context.Context, actor Actor, opts m.ReportOptions) ([]m.OrderDynamicsPoint, error) {
	if !actor.HasRole(m.RoleAdmin, m.RoleAnalyst) {
		return nil, ErrPermissionDenied
	}
	opts, err := normalizeReportOptions(opts)
	if err != nil {
		return nil, err
	}

	points, err := bs.backofficeRepo.GetOrderDynamics(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetOrderDynamics, err)
	}
	return points, nil
}

func (bs BackofficeService) GetSalesByCategory(ctx context.Context, actor Actor, opts m.ReportOptions) ([]m.CategorySalesStats, error) {
	if !actor.HasRole(m.RoleAdmin, m.RoleAnalyst) {
		return nil, ErrPermissionDenied
	}
	opts, err := normalizeReportOptions(opts)
	if err != nil {
		return nil, err
	}

	stats, err := bs.backofficeRepo.GetSalesByCategory(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetSalesByCategory, err)
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

func normalizeReportOptions(opts m.ReportOptions) (m.ReportOptions, error) {
	if opts.Period == "" {
		opts.Period = m.ReportPeriodDay
	}
	switch opts.Period {
	case m.ReportPeriodDay, m.ReportPeriodWeek, m.ReportPeriodMonth:
	default:
		return m.ReportOptions{}, ErrInvalidReportOptions
	}
	if opts.DateFrom != nil && opts.DateTo != nil && opts.DateFrom.After(*opts.DateTo) {
		return m.ReportOptions{}, ErrInvalidReportOptions
	}
	if opts.Limit < 0 {
		return m.ReportOptions{}, ErrInvalidReportOptions
	}
	return opts, nil
}
