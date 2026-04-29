package repository

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	m "github.com/beastixq/marketplace/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BackofficeRepoImpl struct {
	pool *pgxpool.Pool
}

func NewBackofficeRepo(pool *pgxpool.Pool) BackofficeRepoImpl {
	return BackofficeRepoImpl{pool: pool}
}

func (br BackofficeRepoImpl) GetAdminOrders(ctx context.Context, opts m.AdminOrderListOptions) ([]m.Order, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	qb := psql.
		Select("id", "user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at").
		From("orders").
		Where(sq.NotEq{"status": m.StatusDraft}).
		OrderBy("created_at DESC")
	if opts.Status != nil {
		qb = qb.Where(sq.Eq{"status": *opts.Status})
	}
	if opts.Pagination.Page > 0 && opts.Pagination.Limit > 0 {
		qb = qb.
			Offset(uint64((opts.Pagination.Page - 1) * opts.Pagination.Limit)).
			Limit(uint64(opts.Pagination.Limit))
	}

	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := getConn(ctx, br.pool).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()

	orders := make([]m.Order, 0)
	var row orderRow
	for rows.Next() {
		if err = rows.Scan(&row.ID, &row.UserID, &row.AddressID, &row.SellerID, &row.Status, &row.TotalAmount, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		orders = append(orders, row.toModel())
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return orders, nil
}

func (br BackofficeRepoImpl) GetPlatformStats(ctx context.Context) (m.PlatformStats, error) {
	conn := getConn(ctx, br.pool)
	stats := m.PlatformStats{}

	if err := conn.QueryRow(ctx, "SELECT count(*) FROM users WHERE deleted_at IS NULL").Scan(&stats.TotalUsers); err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM sellers").Scan(&stats.TotalSellers); err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM products WHERE deleted_at IS NULL").Scan(&stats.TotalProducts); err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM orders WHERE status != 'draft'").Scan(&stats.TotalOrders); err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM reviews").Scan(&stats.TotalReviews); err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	if err := conn.QueryRow(ctx, "SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status IN ('paid','shipped','delivered')").Scan(&stats.TotalRevenue); err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}

	roleRows, err := conn.Query(ctx, "SELECT role, count(*) FROM users WHERE deleted_at IS NULL GROUP BY role ORDER BY count(*) DESC")
	if err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	for roleRows.Next() {
		var count m.RoleCount
		if err = roleRows.Scan(&count.Role, &count.Count); err != nil {
			roleRows.Close()
			return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		stats.UsersByRole = append(stats.UsersByRole, count)
	}
	roleRows.Close()
	if err = roleRows.Err(); err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}

	statusRows, err := conn.Query(ctx, "SELECT status, count(*) FROM orders GROUP BY status ORDER BY count(*) DESC")
	if err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	for statusRows.Next() {
		var count m.StatusCount
		if err = statusRows.Scan(&count.Status, &count.Count); err != nil {
			statusRows.Close()
			return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		stats.OrdersByStatus = append(stats.OrdersByStatus, count)
	}
	statusRows.Close()
	if err = statusRows.Err(); err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}

	topRows, err := conn.Query(ctx, `
		SELECT p.id, p.name, COALESCE(SUM(oi.price_at_purchase * oi.quantity), 0) AS revenue, COALESCE(SUM(oi.quantity), 0) AS units
		FROM products p
		JOIN order_items oi ON oi.product_id = p.id
		JOIN orders o ON o.id = oi.order_id AND o.status IN ('paid','shipped','delivered')
		GROUP BY p.id, p.name
		ORDER BY revenue DESC
		LIMIT 10
	`)
	if err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	for topRows.Next() {
		var product m.TopProductStats
		if err = topRows.Scan(&product.ID, &product.Name, &product.Revenue, &product.UnitsSold); err != nil {
			topRows.Close()
			return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		stats.TopProducts = append(stats.TopProducts, product)
	}
	topRows.Close()
	if err = topRows.Err(); err != nil {
		return m.PlatformStats{}, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}

	return stats, nil
}

func (br BackofficeRepoImpl) GetOrderDynamics(ctx context.Context, opts m.ReportOptions) ([]m.OrderDynamicsPoint, error) {
	period, err := reportPeriodSQL(opts.Period)
	if err != nil {
		return nil, err
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	qb := psql.
		Select(
			fmt.Sprintf("date_trunc('%s', created_at) AS period_start", period),
			"count(*) AS orders_count",
			"COALESCE(SUM(CASE WHEN status IN ('paid', 'shipped', 'delivered') THEN total_amount ELSE 0 END), 0) AS revenue",
		).
		From("orders").
		Where(sq.NotEq{"status": m.StatusDraft}).
		GroupBy("period_start").
		OrderBy("period_start ASC")
	if opts.DateFrom != nil {
		qb = qb.Where(sq.GtOrEq{"created_at": *opts.DateFrom})
	}
	if opts.DateTo != nil {
		qb = qb.Where(sq.LtOrEq{"created_at": *opts.DateTo})
	}

	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := getConn(ctx, br.pool).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()

	points := make([]m.OrderDynamicsPoint, 0)
	for rows.Next() {
		var point m.OrderDynamicsPoint
		if err = rows.Scan(&point.PeriodStart, &point.OrdersCount, &point.Revenue); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		points = append(points, point)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return points, nil
}

func (br BackofficeRepoImpl) GetSalesByCategory(ctx context.Context, opts m.ReportOptions) ([]m.CategorySalesStats, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	qb := psql.
		Select(
			"c.id",
			"c.name",
			"count(DISTINCT o.id) AS orders_count",
			"COALESCE(SUM(oi.quantity), 0) AS units_sold",
			"COALESCE(SUM(oi.price_at_purchase * oi.quantity), 0) AS revenue",
		).
		From("categories c").
		Join("product_categories pc ON pc.category_id = c.id").
		Join("order_items oi ON oi.product_id = pc.product_id").
		Join("orders o ON o.id = oi.order_id AND o.status IN ('paid', 'shipped', 'delivered')").
		GroupBy("c.id", "c.name").
		OrderBy("revenue DESC", "c.name ASC")
	if opts.DateFrom != nil {
		qb = qb.Where(sq.GtOrEq{"o.created_at": *opts.DateFrom})
	}
	if opts.DateTo != nil {
		qb = qb.Where(sq.LtOrEq{"o.created_at": *opts.DateTo})
	}
	if opts.Limit > 0 {
		qb = qb.Limit(uint64(opts.Limit))
	}

	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := getConn(ctx, br.pool).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()

	stats := make([]m.CategorySalesStats, 0)
	for rows.Next() {
		var stat m.CategorySalesStats
		if err = rows.Scan(&stat.CategoryID, &stat.CategoryName, &stat.OrdersCount, &stat.UnitsSold, &stat.Revenue); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		stats = append(stats, stat)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return stats, nil
}

func reportPeriodSQL(period m.ReportPeriod) (string, error) {
	switch period {
	case "", m.ReportPeriodDay:
		return "day", nil
	case m.ReportPeriodWeek:
		return "week", nil
	case m.ReportPeriodMonth:
		return "month", nil
	default:
		return "", fmt.Errorf("invalid report period: %s", period)
	}
}
