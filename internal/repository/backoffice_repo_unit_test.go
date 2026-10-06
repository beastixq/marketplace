package repository_test

import (
	"context"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/repository"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/shopspring/decimal"
)

func TestBackofficeRepo_GetAdminOrders(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM orders WHERE status <> \\$1 ORDER BY created_at DESC").WithArgs(m.StatusDraft).WillReturnRows(backofficeOrderRows()).RowsWillBeClosed()
		got, err := repository.NewBackofficeRepo(db).GetAdminOrders(ctx, m.AdminOrderListOptions{})
		requireErrorIs(t, err, nil)
		if len(got) != 1 {
			t.Fatalf("unexpected orders: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM orders WHERE status <> \\$1 ORDER BY created_at DESC").WithArgs(m.StatusDraft).WillReturnError(unitErr)
		_, err := repository.NewBackofficeRepo(db).GetAdminOrders(ctx, m.AdminOrderListOptions{})
		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at"}).AddRow("invalid ID", int64(2), nil, nil, string(m.StatusPaid), decimal.NewFromInt(100), unitNow, unitNow), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: backofficeOrderRows().CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM orders WHERE status <> \\$1 ORDER BY created_at DESC").WithArgs(m.StatusDraft).WillReturnRows(tc.rows).RowsWillBeClosed()
			_, err := repository.NewBackofficeRepo(db).GetAdminOrders(ctx, m.AdminOrderListOptions{})
			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestBackofficeRepo_GetPlatformStats(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		for _, query := range []string{
			"SELECT count\\(\\*\\) FROM users WHERE deleted_at IS NULL",
			"SELECT count\\(\\*\\) FROM sellers",
			"SELECT count\\(\\*\\) FROM products WHERE deleted_at IS NULL",
			"SELECT count\\(\\*\\) FROM orders WHERE status != 'draft'",
			"SELECT count\\(\\*\\) FROM reviews",
		} {
			db.ExpectQuery(query).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(int64(1)))
		}
		db.ExpectQuery("SELECT COALESCE\\(SUM\\(total_amount\\), 0\\) FROM orders (.+)").WillReturnRows(pgxmock.NewRows([]string{"sum"}).AddRow(decimal.NewFromInt(100)))
		db.ExpectQuery("SELECT role, count\\(\\*\\) FROM users (.+)").WillReturnRows(pgxmock.NewRows([]string{"role", "count"}).AddRow(m.RoleBuyer, int64(1))).RowsWillBeClosed()
		db.ExpectQuery("SELECT status, count\\(\\*\\) FROM orders (.+)").WillReturnRows(pgxmock.NewRows([]string{"status", "count"}).AddRow(m.StatusPaid, int64(1))).RowsWillBeClosed()
		db.ExpectQuery("(?s)SELECT p.id, p.name, COALESCE.+").WillReturnRows(pgxmock.NewRows([]string{"id", "name", "revenue", "units"}).AddRow(int64(1), "Product", decimal.NewFromInt(100), int64(1))).RowsWillBeClosed()

		got, err := repository.NewBackofficeRepo(db).GetPlatformStats(ctx)

		requireErrorIs(t, err, nil)
		if got.TotalUsers != 1 || len(got.TopProducts) != 1 {
			t.Fatalf("unexpected platform stats: %+v", got)
		}
	})
	t.Run("first query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT count\\(\\*\\) FROM users WHERE deleted_at IS NULL").WillReturnError(unitErr)
		_, err := repository.NewBackofficeRepo(db).GetPlatformStats(ctx)
		requireErrorIs(t, err, repository.ErrToScan)
	})
}

func TestBackofficeRepo_GetOrderDynamics(t *testing.T) {
	ctx := context.Background()

	opts := m.ReportOptions{Period: m.ReportPeriodDay}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT date_trunc\\('day', created_at\\) AS period_start, (.+) FROM orders WHERE status <> \\$1 GROUP BY period_start ORDER BY period_start ASC").WithArgs(m.StatusDraft).WillReturnRows(pgxmock.NewRows([]string{"period_start", "orders_count", "revenue"}).AddRow(unitNow, int64(1), decimal.NewFromInt(100))).RowsWillBeClosed()
		got, err := repository.NewBackofficeRepo(db).GetOrderDynamics(ctx, opts)
		requireErrorIs(t, err, nil)
		if len(got) != 1 {
			t.Fatalf("unexpected dynamics: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT date_trunc\\('day', created_at\\) AS period_start, (.+) FROM orders WHERE status <> \\$1 GROUP BY period_start ORDER BY period_start ASC").WithArgs(m.StatusDraft).WillReturnError(unitErr)
		_, err := repository.NewBackofficeRepo(db).GetOrderDynamics(ctx, opts)
		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"invalid"}).AddRow("invalid"), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: pgxmock.NewRows([]string{"period_start", "orders_count", "revenue"}).AddRow(unitNow, int64(1), decimal.NewFromInt(100)).CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT date_trunc\\('day', created_at\\) AS period_start, (.+) FROM orders WHERE status <> \\$1 GROUP BY period_start ORDER BY period_start ASC").WithArgs(m.StatusDraft).WillReturnRows(tc.rows).RowsWillBeClosed()
			_, err := repository.NewBackofficeRepo(db).GetOrderDynamics(ctx, opts)
			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestBackofficeRepo_GetSalesByCategory(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM categories c JOIN (.+) GROUP BY c.id, c.name ORDER BY revenue DESC, c.name ASC").WillReturnRows(pgxmock.NewRows([]string{"category_id", "category_name", "orders_count", "units_sold", "revenue"}).AddRow(int64(1), "Category", int64(1), int64(2), decimal.NewFromInt(100))).RowsWillBeClosed()
		got, err := repository.NewBackofficeRepo(db).GetSalesByCategory(ctx, m.ReportOptions{})
		requireErrorIs(t, err, nil)
		if len(got) != 1 {
			t.Fatalf("unexpected category stats: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM categories c JOIN (.+) GROUP BY c.id, c.name ORDER BY revenue DESC, c.name ASC").WillReturnError(unitErr)
		_, err := repository.NewBackofficeRepo(db).GetSalesByCategory(ctx, m.ReportOptions{})
		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"invalid"}).AddRow("invalid"), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: pgxmock.NewRows([]string{"category_id", "category_name", "orders_count", "units_sold", "revenue"}).AddRow(int64(1), "Category", int64(1), int64(2), decimal.NewFromInt(100)).CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM categories c JOIN (.+) GROUP BY c.id, c.name ORDER BY revenue DESC, c.name ASC").WillReturnRows(tc.rows).RowsWillBeClosed()
			_, err := repository.NewBackofficeRepo(db).GetSalesByCategory(ctx, m.ReportOptions{})
			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func backofficeOrderRows() *pgxmock.Rows {
	orderColumns := []string{"id", "user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at"}
	return pgxmock.NewRows(orderColumns).AddRow(int64(1), int64(2), nil, nil, string(m.StatusPaid), decimal.NewFromInt(100), unitNow, unitNow)
}
