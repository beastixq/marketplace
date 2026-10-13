//go:build integration

package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	s "github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
)

func TestBackofficeService_GetPlatformStats_Integration(t *testing.T) {
	t.Run("aggregate paid order filters and role boundaries", func(t *testing.T) {
		f := newIntegrationFixture(t)
		ctx := context.Background()
		backoffice := s.NewBackofficeService(repo.NewBackofficeRepo(f.pool))
		analyst, admin := s.Actor{Role: m.RoleAnalyst}, s.Actor{Role: m.RoleAdmin}
		before, err := backoffice.GetPlatformStats(ctx, analyst)
		mustIntegration(t, err)
		buyer := f.actor(m.RoleBuyer)
		seller, sellerID := f.seller()
		productID, categoryID := f.product(seller, sellerID, 2), f.category()
		mustIntegration(t, f.products.ReplaceProductCategories(ctx, seller, productID, []int64{categoryID}))
		orderID := f.checkout(buyer, productID, f.address(buyer))
		mustIntegration(t, f.orders.PayOrder(ctx, buyer, orderID))

		after, err := backoffice.GetPlatformStats(ctx, analyst)
		mustIntegration(t, err)
		if after.TotalUsers-before.TotalUsers != 2 || after.TotalSellers-before.TotalSellers != 1 || after.TotalProducts-before.TotalProducts != 1 || after.TotalOrders-before.TotalOrders != 1 || !after.TotalRevenue.Sub(before.TotalRevenue).Equal(decimal.NewFromInt(100)) {
			t.Fatalf("aggregate delta: before=%+v after=%+v", before, after)
		}
		paid := m.StatusPaid
		orders, err := backoffice.GetAdminOrders(ctx, admin, m.AdminOrderListOptions{Status: &paid})
		mustIntegration(t, err)
		found := false
		for _, order := range orders {
			if order.Status != paid {
				t.Fatal("status filter ignored")
			}
			found = found || order.ID == orderID
		}
		if !found {
			t.Fatal("own paid order missing")
		}
		if _, err := backoffice.GetAdminOrders(ctx, analyst, m.AdminOrderListOptions{}); !errors.Is(err, s.ErrPermissionDenied) {
			t.Fatalf("analyst writes boundary = %v", err)
		}
		if _, err := backoffice.GetPlatformStats(ctx, buyer); !errors.Is(err, s.ErrPermissionDenied) {
			t.Fatalf("buyer report = %v", err)
		}
		from := time.Now().UTC().Truncate(24 * time.Hour)
		to := from.AddDate(0, 0, 1)
		stats, err := f.sellers.GetSellerStats(ctx, sellerID, from, to)
		mustIntegration(t, err)
		if stats.TotalOrders != 1 || !stats.TotalRevenue.Equal(decimal.NewFromInt(100)) || !stats.AvgOrderValue.Equal(decimal.NewFromInt(100)) || stats.TopProductName != "Integration product" {
			t.Fatalf("seller stats = %+v", stats)
		}
		categories, err := backoffice.GetSalesByCategory(ctx, analyst, m.ReportOptions{DateFrom: &from, DateTo: &to})
		mustIntegration(t, err)
		found = false
		for _, category := range categories {
			if category.CategoryID == categoryID {
				found = true
				if category.OrdersCount != 1 || category.UnitsSold != 1 || !category.Revenue.Equal(decimal.NewFromInt(100)) {
					t.Fatalf("category aggregate = %+v", category)
				}
			}
		}
		if !found {
			t.Fatal("own category missing in report")
		}
		_, err = backoffice.GetOrderDynamics(ctx, analyst, m.ReportOptions{DateFrom: &to, DateTo: &from})
		if !errors.Is(err, s.ErrInvalidReportOptions) {
			t.Fatalf("date filter = %v", err)
		}
	})
}
