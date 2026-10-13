//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/shopspring/decimal"
)

func TestBackofficeRepo_GetOrderDynamics_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	t.Run("dates periods and paid revenue", func(t *testing.T) {
		ctx := context.Background()
		r := repo.NewBackofficeRepo(testPool)
		from := time.Date(2040, 2, 3, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 0, 1)
		options := m.ReportOptions{DateFrom: &from, DateTo: &to, Period: m.ReportPeriodDay}
		before, err := r.GetOrderDynamics(ctx, options)
		if err != nil {
			t.Fatal(err)
		}
		buyerID, sellerID := createTestUser(t), createTestSeller(t)
		orderID := createTestOrder(t, buyerID, sellerID)
		if _, err := testPool.Exec(ctx, "UPDATE orders SET status='paid', total_amount=250, created_at=$1 WHERE id=$2", from.Add(time.Hour), orderID); err != nil {
			t.Fatal(err)
		}

		points, err := r.GetOrderDynamics(ctx, options)
		if err != nil {
			t.Fatal(err)
		}
		var beforeCount, afterCount int64
		beforeRevenue, afterRevenue := decimal.Zero, decimal.Zero
		for _, point := range before {
			beforeCount += point.OrdersCount
			beforeRevenue = beforeRevenue.Add(point.Revenue)
		}
		for _, point := range points {
			afterCount += point.OrdersCount
			afterRevenue = afterRevenue.Add(point.Revenue)
		}
		if afterCount-beforeCount != 1 || !afterRevenue.Sub(beforeRevenue).Equal(decimal.NewFromInt(250)) {
			t.Fatalf("dynamics = %+v", points)
		}
		for _, period := range []m.ReportPeriod{m.ReportPeriodWeek, m.ReportPeriodMonth} {
			options.Period = period
			if _, err := r.GetOrderDynamics(ctx, options); err != nil {
				t.Fatalf("period %s: %v", period, err)
			}
		}
		options.DateFrom = &to
		excluded, err := r.GetOrderDynamics(ctx, options)
		if err != nil {
			t.Fatal(err)
		}
		if len(excluded) != 0 {
			t.Fatalf("date exclusion = %+v", excluded)
		}
	})
}

func TestBackofficeRepo_GetAdminOrders_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	t.Run("draft hidden status filtered pagination", func(t *testing.T) {
		ctx := context.Background()
		buyerID, sellerID := createTestUser(t), createTestSeller(t)
		id := createTestOrder(t, buyerID, sellerID)
		orders := repo.NewBackofficeRepo(testPool)
		before, err := orders.GetAdminOrders(ctx, m.AdminOrderListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		for _, order := range before {
			if order.ID == id {
				t.Fatal("draft visible")
			}
		}
		if _, err := testPool.Exec(ctx, "UPDATE orders SET status='paid' WHERE id=$1", id); err != nil {
			t.Fatal(err)
		}

		paid := m.StatusPaid
		after, err := orders.GetAdminOrders(ctx, m.AdminOrderListOptions{Status: &paid})
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, order := range after {
			if order.Status != paid {
				t.Fatal("filter ignored")
			}
			found = found || order.ID == id
		}
		if !found {
			t.Fatal("paid fixture missing")
		}
		page, err := orders.GetAdminOrders(ctx, m.AdminOrderListOptions{Status: &paid, Pagination: m.PaginationOpts{Page: 1, Limit: 1}})
		if err != nil {
			t.Fatal(err)
		}
		if len(page) != 1 {
			t.Fatalf("page size = %d", len(page))
		}
	})
}
