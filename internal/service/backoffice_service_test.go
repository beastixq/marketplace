package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
)

type fakeBackofficeRepo struct {
	orders          []m.Order
	stats           m.PlatformStats
	dynamics        []m.OrderDynamicsPoint
	salesByCategory []m.CategorySalesStats
	err             error
	called          bool
	reportOpts      m.ReportOptions
}

func (r *fakeBackofficeRepo) GetAdminOrders(context.Context, m.AdminOrderListOptions) ([]m.Order, error) {
	r.called = true
	return r.orders, r.err
}

func (r *fakeBackofficeRepo) GetPlatformStats(context.Context) (m.PlatformStats, error) {
	r.called = true
	return r.stats, r.err
}

func (r *fakeBackofficeRepo) GetOrderDynamics(_ context.Context, opts m.ReportOptions) ([]m.OrderDynamicsPoint, error) {
	r.called = true
	r.reportOpts = opts
	return r.dynamics, r.err
}

func (r *fakeBackofficeRepo) GetSalesByCategory(_ context.Context, opts m.ReportOptions) ([]m.CategorySalesStats, error) {
	r.called = true
	r.reportOpts = opts
	return r.salesByCategory, r.err
}

func TestBackofficeGetAdminOrders(t *testing.T) {
	ctx := context.Background()

	tCases := []struct {
		description string
		actor       service.Actor
		status      *m.OrderStatus
		repoErr     error
		wantErr     error
		wantCalled  bool
	}{
		{
			description: "admin can list orders",
			actor:       service.Actor{UserID: someID, Role: m.RoleAdmin},
			wantCalled:  true,
		},
		{
			description: "analyst cannot list admin orders",
			actor:       service.Actor{UserID: someID, Role: m.RoleAnalyst},
			wantErr:     service.ErrPermissionDenied,
		},
		{
			description: "invalid status rejected before repo",
			actor:       service.Actor{UserID: someID, Role: m.RoleAdmin},
			status:      ptr(m.StatusDraft),
			wantErr:     service.ErrOrderStatusInvalid,
		},
		{
			description: "repo error wrapped",
			actor:       service.Actor{UserID: someID, Role: m.RoleAdmin},
			repoErr:     errors.New("repo failed"),
			wantErr:     service.ErrGetAdminOrders,
			wantCalled:  true,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.description, func(t *testing.T) {
			repo := &fakeBackofficeRepo{err: tCase.repoErr}
			svc := service.NewBackofficeService(repo)
			_, err := svc.GetAdminOrders(ctx, tCase.actor, m.AdminOrderListOptions{Status: tCase.status})
			assertError(t, err, tCase.wantErr)
			if repo.called != tCase.wantCalled {
				t.Fatalf("invalid repo call state. expected: %v, got: %v", tCase.wantCalled, repo.called)
			}
		})
	}
}

func TestBackofficeGetPlatformStats(t *testing.T) {
	ctx := context.Background()

	tCases := []struct {
		description string
		actor       service.Actor
		repoErr     error
		wantErr     error
		wantCalled  bool
	}{
		{
			description: "admin can view platform stats",
			actor:       service.Actor{UserID: someID, Role: m.RoleAdmin},
			wantCalled:  true,
		},
		{
			description: "analyst can view platform stats",
			actor:       service.Actor{UserID: someID, Role: m.RoleAnalyst},
			wantCalled:  true,
		},
		{
			description: "buyer cannot view platform stats",
			actor:       service.Actor{UserID: someID, Role: m.RoleBuyer},
			wantErr:     service.ErrPermissionDenied,
		},
		{
			description: "repo error wrapped",
			actor:       service.Actor{UserID: someID, Role: m.RoleAdmin},
			repoErr:     errors.New("repo failed"),
			wantErr:     service.ErrGetPlatformStats,
			wantCalled:  true,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.description, func(t *testing.T) {
			repo := &fakeBackofficeRepo{err: tCase.repoErr}
			svc := service.NewBackofficeService(repo)
			_, err := svc.GetPlatformStats(ctx, tCase.actor)
			assertError(t, err, tCase.wantErr)
			if repo.called != tCase.wantCalled {
				t.Fatalf("invalid repo call state. expected: %v, got: %v", tCase.wantCalled, repo.called)
			}
		})
	}
}

func TestBackofficeGetOrderDynamics(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	earlier := now.Add(-24 * time.Hour)

	tCases := []struct {
		description string
		actor       service.Actor
		opts        m.ReportOptions
		repoErr     error
		wantErr     error
		wantCalled  bool
		wantPeriod  m.ReportPeriod
	}{
		{
			description: "analyst can view order dynamics",
			actor:       service.Actor{UserID: someID, Role: m.RoleAnalyst},
			opts:        m.ReportOptions{DateFrom: &earlier, DateTo: &now},
			wantCalled:  true,
			wantPeriod:  m.ReportPeriodDay,
		},
		{
			description: "admin can view weekly order dynamics",
			actor:       service.Actor{UserID: someID, Role: m.RoleAdmin},
			opts:        m.ReportOptions{Period: m.ReportPeriodWeek},
			wantCalled:  true,
			wantPeriod:  m.ReportPeriodWeek,
		},
		{
			description: "buyer cannot view order dynamics",
			actor:       service.Actor{UserID: someID, Role: m.RoleBuyer},
			wantErr:     service.ErrPermissionDenied,
		},
		{
			description: "invalid period rejected before repo",
			actor:       service.Actor{UserID: someID, Role: m.RoleAnalyst},
			opts:        m.ReportOptions{Period: m.ReportPeriod("year")},
			wantErr:     service.ErrInvalidReportOptions,
		},
		{
			description: "invalid date range rejected before repo",
			actor:       service.Actor{UserID: someID, Role: m.RoleAnalyst},
			opts:        m.ReportOptions{DateFrom: &now, DateTo: &earlier},
			wantErr:     service.ErrInvalidReportOptions,
		},
		{
			description: "repo error wrapped",
			actor:       service.Actor{UserID: someID, Role: m.RoleAnalyst},
			repoErr:     errors.New("repo failed"),
			wantErr:     service.ErrGetOrderDynamics,
			wantCalled:  true,
			wantPeriod:  m.ReportPeriodDay,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.description, func(t *testing.T) {
			repo := &fakeBackofficeRepo{err: tCase.repoErr}
			svc := service.NewBackofficeService(repo)
			_, err := svc.GetOrderDynamics(ctx, tCase.actor, tCase.opts)
			assertError(t, err, tCase.wantErr)
			if repo.called != tCase.wantCalled {
				t.Fatalf("invalid repo call state. expected: %v, got: %v", tCase.wantCalled, repo.called)
			}
			if tCase.wantCalled && repo.reportOpts.Period != tCase.wantPeriod {
				t.Fatalf("invalid report period. expected: %q, got: %q", tCase.wantPeriod, repo.reportOpts.Period)
			}
		})
	}
}

func TestBackofficeGetSalesByCategory(t *testing.T) {
	ctx := context.Background()

	tCases := []struct {
		description string
		actor       service.Actor
		opts        m.ReportOptions
		repoErr     error
		wantErr     error
		wantCalled  bool
	}{
		{
			description: "analyst can view category sales",
			actor:       service.Actor{UserID: someID, Role: m.RoleAnalyst},
			opts:        m.ReportOptions{Limit: 10},
			wantCalled:  true,
		},
		{
			description: "admin can view category sales",
			actor:       service.Actor{UserID: someID, Role: m.RoleAdmin},
			wantCalled:  true,
		},
		{
			description: "buyer cannot view category sales",
			actor:       service.Actor{UserID: someID, Role: m.RoleBuyer},
			wantErr:     service.ErrPermissionDenied,
		},
		{
			description: "negative limit rejected before repo",
			actor:       service.Actor{UserID: someID, Role: m.RoleAnalyst},
			opts:        m.ReportOptions{Limit: -1},
			wantErr:     service.ErrInvalidReportOptions,
		},
		{
			description: "repo error wrapped",
			actor:       service.Actor{UserID: someID, Role: m.RoleAnalyst},
			repoErr:     errors.New("repo failed"),
			wantErr:     service.ErrGetSalesByCategory,
			wantCalled:  true,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.description, func(t *testing.T) {
			repo := &fakeBackofficeRepo{err: tCase.repoErr}
			svc := service.NewBackofficeService(repo)
			_, err := svc.GetSalesByCategory(ctx, tCase.actor, tCase.opts)
			assertError(t, err, tCase.wantErr)
			if repo.called != tCase.wantCalled {
				t.Fatalf("invalid repo call state. expected: %v, got: %v", tCase.wantCalled, repo.called)
			}
		})
	}
}

func ptr[T any](value T) *T {
	return &value
}
