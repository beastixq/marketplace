package service_test

import (
	"context"
	"errors"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
)

type fakeBackofficeRepo struct {
	orders []m.Order
	stats  m.PlatformStats
	err    error
	called bool
}

func (r *fakeBackofficeRepo) GetAdminOrders(context.Context, m.AdminOrderListOptions) ([]m.Order, error) {
	r.called = true
	return r.orders, r.err
}

func (r *fakeBackofficeRepo) GetPlatformStats(context.Context) (m.PlatformStats, error) {
	r.called = true
	return r.stats, r.err
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

func ptr[T any](value T) *T {
	return &value
}
