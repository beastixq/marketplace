package repository

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepoImpl struct {
	pool *pgxpool.Pool
}

func NewOrderRepo(pool *pgxpool.Pool) OrderRepoImpl {
	return OrderRepoImpl{pool: pool}
}

func (or OrderRepoImpl) GetOrderByID(ctx context.Context, id int64) (o m.Order, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at").From("orders").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return m.Order{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := getConn(ctx, or.pool).QueryRow(ctx, sql, args...)
	var order orderRow
	if err = row.Scan(&order.ID, &order.UserID, &order.AddressID, &order.SellerID, &order.Status, &order.TotalAmount, &order.CreatedAt, &order.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.Order{}, service.ErrNotFound
		}
		return m.Order{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return order.toModel(), nil
}

func (or OrderRepoImpl) GetOrdersByUserID(ctx context.Context, userID int64) (orders []m.Order, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at").From("orders").Where(sq.Eq{"user_id": userID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := getConn(ctx, or.pool).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()
	orders = make([]m.Order, 0)
	var orow orderRow
	for rows.Next() {
		err = rows.Scan(&orow.ID, &orow.UserID, &orow.AddressID, &orow.SellerID, &orow.Status, &orow.TotalAmount, &orow.CreatedAt, &orow.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		orders = append(orders, orow.toModel())
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return orders, nil
}

func (or OrderRepoImpl) CreateOrder(ctx context.Context, oc m.OrderCreate) (id int64, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Insert("orders").Columns("user_id", "address_id", "seller_id", "status", "total_amount").Values(oc.UserID, oc.AddressID, oc.SellerID, oc.Status, oc.TotalAmount).Suffix("RETURNING id").ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := getConn(ctx, or.pool).QueryRow(ctx, sql, args...)
	if err = row.Scan(&id); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return id, nil
}

func (or OrderRepoImpl) UpdateOrder(ctx context.Context, id int64, ou m.OrderUpdate) (o m.Order, err error) {
	if ou.UserID == nil && ou.AddressID == nil && ou.Status == nil && ou.TotalAmount == nil {
		return m.Order{}, service.ErrNoChangesInUpdate
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("orders").Where(sq.Eq{"id": id})
	if ou.UserID != nil {
		ub = ub.Set("user_id", *ou.UserID)
	}
	if ou.AddressID != nil {
		ub = ub.Set("address_id", *ou.AddressID)
	}
	if ou.Status != nil {
		ub = ub.Set("status", *ou.Status)
	}
	if ou.TotalAmount != nil {
		ub = ub.Set("total_amount", *ou.TotalAmount)
	}
	ub = ub.Suffix("RETURNING id, user_id, address_id, seller_id, status, total_amount, created_at, updated_at")
	sql, args, err := ub.ToSql()
	if err != nil {
		return m.Order{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := getConn(ctx, or.pool).QueryRow(ctx, sql, args...)
	var order orderRow
	if err = row.Scan(&order.ID, &order.UserID, &order.AddressID, &order.SellerID, &order.Status, &order.TotalAmount, &order.CreatedAt, &order.UpdatedAt); err != nil {
		return m.Order{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return order.toModel(), nil
}

func (or OrderRepoImpl) DeleteOrderByID(ctx context.Context, id int64) (err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.
		Delete("orders").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	if _, err = getConn(ctx, or.pool).Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}

func (or OrderRepoImpl) GetSellerOrdersBySellerID(ctx context.Context, sellerID int64, pg m.PaginationOpts) (orders []m.Order, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	qb := psql.
		Select("id", "user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at").
		From("orders").
		Where(sq.Eq{"seller_id": sellerID}).
		OrderBy("created_at DESC")
	if pg.Page > 0 && pg.Limit > 0 {
		qb = qb.Offset(uint64((pg.Page - 1) * pg.Limit)).Limit(uint64(pg.Limit))
	}
	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := getConn(ctx, or.pool).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()
	orders = make([]m.Order, 0)
	var orow orderRow
	for rows.Next() {
		err = rows.Scan(&orow.ID, &orow.UserID, &orow.AddressID, &orow.SellerID, &orow.Status, &orow.TotalAmount, &orow.CreatedAt, &orow.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		orders = append(orders, orow.toModel())
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return orders, nil
}
