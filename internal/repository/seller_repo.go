package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SellerRepoImpl struct {
	pool *pgxpool.Pool
}

func NewSellerRepo(pool *pgxpool.Pool) SellerRepoImpl {
	return SellerRepoImpl{pool: pool}
}

func (sr SellerRepoImpl) GetSellerByID(ctx context.Context, id int64) (s m.Seller, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "user_id", "company_name", "description", "rating", "created_at").From("sellers").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return m.Seller{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := sr.pool.QueryRow(ctx, sql, args...)
	var seller sellerRow
	if err = row.Scan(&seller.ID, &seller.UserID, &seller.CompanyName, &seller.Description, &seller.Rating, &seller.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.Seller{}, service.ErrNotFound
		}
		return m.Seller{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return seller.toModel(), nil
}

func (sr SellerRepoImpl) GetSellerByUserID(ctx context.Context, userID int64) (s m.Seller, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "user_id", "company_name", "description", "rating", "created_at").From("sellers").Where(sq.Eq{"user_id": userID}).ToSql()
	if err != nil {
		return m.Seller{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := sr.pool.QueryRow(ctx, sql, args...)
	var seller sellerRow
	if err = row.Scan(&seller.ID, &seller.UserID, &seller.CompanyName, &seller.Description, &seller.Rating, &seller.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.Seller{}, service.ErrNotFound
		}
		return m.Seller{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return seller.toModel(), nil
}

func (sr SellerRepoImpl) GetSellerStats(ctx context.Context, sellerID int64, dateFrom time.Time, dateTo time.Time) (ss m.SellerStats, err error) {
	sql := "SELECT total_orders, total_revenue, avg_order_value, top_product_name FROM get_seller_statistics($1, $2, $3)"
	row := sr.pool.QueryRow(ctx, sql, sellerID, dateFrom, dateTo)
	var ssrow sellerStatsRow
	if err = row.Scan(&ssrow.TotalOrders, &ssrow.TotalRevenue, &ssrow.AvgOrderValue, &ssrow.TopProductName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.SellerStats{}, service.ErrNotFound
		}
		return m.SellerStats{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return ssrow.toModel(), nil
}

func (sr SellerRepoImpl) CreateSeller(ctx context.Context, sc m.SellerCreate) (id int64, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.
		Insert("sellers").
		Columns("user_id", "company_name", "description", "rating").
		Values(sc.UserID, sc.CompanyName, sc.Description, sc.Rating).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := sr.pool.QueryRow(ctx, sql, args...)
	if err = row.Scan(&id); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return id, nil
}

func (sr SellerRepoImpl) UpdateSeller(ctx context.Context, id int64, su m.SellerUpdate) (s m.Seller, err error) {
	if su.UserID == nil && su.CompanyName == nil && su.Description == nil && su.Rating == nil {
		return m.Seller{}, service.ErrNoChangesInUpdate
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("sellers").Where(sq.Eq{"id": id})
	if su.UserID != nil {
		ub = ub.Set("user_id", *su.UserID)
	}
	if su.CompanyName != nil {
		ub = ub.Set("company_name", *su.CompanyName)
	}
	if su.Description != nil {
		ub = ub.Set("description", *su.Description)
	}
	if su.Rating != nil {
		ub = ub.Set("rating", *su.Rating)
	}
	ub = ub.Suffix("RETURNING id, user_id, company_name, description, rating, created_at")
	sql, args, err := ub.ToSql()
	if err != nil {
		return m.Seller{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := sr.pool.QueryRow(ctx, sql, args...)
	var seller sellerRow
	if err = row.Scan(&seller.ID, &seller.UserID, &seller.CompanyName, &seller.Description, &seller.Rating, &seller.CreatedAt); err != nil {
		return m.Seller{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return seller.toModel(), nil
}

func (sr SellerRepoImpl) DeleteSellerByID(ctx context.Context, id int64) (err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Delete("sellers").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	if _, err = sr.pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}
