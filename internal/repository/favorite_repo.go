package repository

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ service.FavoriteRepo = (*FavoriteRepoImpl)(nil)

type FavoriteRepoImpl struct {
	pool *pgxpool.Pool
}

func NewFavoriteRepo(pool *pgxpool.Pool) FavoriteRepoImpl {
	return FavoriteRepoImpl{pool: pool}
}

// AddFavorite implements service.FavoriteRepo.AddFavorite using
// INSERT ... ON CONFLICT (user_id, product_id) DO NOTHING. RowsAffected
// distinguishes a fresh insert from a conflict no-op; an FK violation maps
// to service.ErrNotFound.
func (fr FavoriteRepoImpl) AddFavorite(ctx context.Context, userID int64, productID int64) (bool, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.
		Insert("product_favorites").
		Columns("user_id", "product_id").
		Values(userID, productID).
		Suffix("ON CONFLICT (user_id, product_id) DO NOTHING").
		ToSql()
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	tag, err := getConn(ctx, fr.pool).Exec(ctx, sql, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			return false, service.ErrNotFound
		}
		return false, fmt.Errorf("%w: %v", ErrExec, err)
	}
	return tag.RowsAffected() == 1, nil
}

func (fr FavoriteRepoImpl) DeleteFavorite(ctx context.Context, userID int64, productID int64) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.
		Delete("product_favorites").
		Where(sq.Eq{"user_id": userID, "product_id": productID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	if _, err = getConn(ctx, fr.pool).Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}

func (fr FavoriteRepoImpl) ListFavoriteProductsByUserID(ctx context.Context, userID int64, opts m.PaginationOpts) ([]m.Product, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sb := psql.
		Select("p.id", "p.seller_id", "p.name", "p.description", "p.price", "p.stock_quantity", "p.reserved_quantity", "p.rating", "p.created_at", "p.deleted_at").
		From("product_favorites pf").
		Join("products p ON p.id = pf.product_id").
		Where(sq.Eq{"pf.user_id": userID}).
		Where(sq.Eq{"p.deleted_at": nil}).
		OrderBy("pf.created_at DESC", "pf.product_id DESC")
	if opts.Page > 0 && opts.Limit > 0 {
		sb = sb.Offset(uint64(opts.Limit * (opts.Page - 1))).Limit(uint64(opts.Limit))
	}
	sql, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := getConn(ctx, fr.pool).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()

	products := make([]m.Product, 0)
	for rows.Next() {
		var row productRow
		err = rows.Scan(&row.ID, &row.SellerID, &row.Name, &row.Description, &row.Price, &row.StockQuantity, &row.ReservedQuantity, &row.Rating, &row.CreatedAt, &row.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		products = append(products, row.toModel())
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return products, nil
}

func (fr FavoriteRepoImpl) IsFavorite(ctx context.Context, userID int64, productID int64) (bool, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.
		Select("1").
		From("product_favorites").
		Where(sq.Eq{"user_id": userID, "product_id": productID}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	var one int
	if err := getConn(ctx, fr.pool).QueryRow(ctx, sql, args...).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return true, nil
}
