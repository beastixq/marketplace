package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FavoriteRepoImpl struct {
	pool *pgxpool.Pool
}

func NewFavoriteRepo(pool *pgxpool.Pool) FavoriteRepoImpl {
	return FavoriteRepoImpl{pool: pool}
}

var _ service.FavoriteRepo = FavoriteRepoImpl{}

// Add inserts (user_id, product_id) into favorites. ON CONFLICT DO NOTHING
// makes the operation idempotent at the SQL level. Returns created=true when
// a row was actually inserted, false when it already existed.
//
// A foreign-key violation on the product side is translated to
// service.ErrProductNotFound — this covers the race where the product is
// deleted between the service-layer visibility check and the INSERT.
func (fr FavoriteRepoImpl) Add(ctx context.Context, userID, productID int64) (created bool, err error) {
	const sql = `
INSERT INTO favorites (user_id, product_id)
VALUES ($1, $2)
ON CONFLICT (user_id, product_id) DO NOTHING`
	tag, err := getConn(ctx, fr.pool).Exec(ctx, sql, userID, productID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			return false, service.ErrProductNotFound
		}
		return false, fmt.Errorf("%w: %v", ErrExec, err)
	}
	return tag.RowsAffected() == 1, nil
}

// Remove deletes the (user_id, product_id) row. Idempotent: returns nil even
// when no row matched.
func (fr FavoriteRepoImpl) Remove(ctx context.Context, userID, productID int64) error {
	const sql = `DELETE FROM favorites WHERE user_id = $1 AND product_id = $2`
	if _, err := getConn(ctx, fr.pool).Exec(ctx, sql, userID, productID); err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}

// List returns a paginated slice of favorites for the user, newest first,
// joined with the current product state. Soft-deleted products remain in
// the response (the caller's DTO surfaces in_stock=false in that case).
func (fr FavoriteRepoImpl) List(ctx context.Context, userID int64, offset, limit int) (items []m.Favorite, total int, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	listSQL, listArgs, err := psql.
		Select(
			"p.id", "p.seller_id", "p.name", "p.description",
			"p.price", "p.stock_quantity", "p.reserved_quantity",
			"p.rating", "p.created_at", "p.deleted_at",
			"f.created_at",
		).
		From("favorites f").
		Join("products p ON p.id = f.product_id").
		Where(sq.Eq{"f.user_id": userID}).
		OrderBy("f.created_at DESC", "f.product_id ASC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrToSql, err)
	}

	rows, err := getConn(ctx, fr.pool).Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()

	items = make([]m.Favorite, 0)
	for rows.Next() {
		var prow productRow
		var addedAt time.Time
		if err = rows.Scan(
			&prow.ID, &prow.SellerID, &prow.Name, &prow.Description,
			&prow.Price, &prow.StockQuantity, &prow.ReservedQuantity,
			&prow.Rating, &prow.CreatedAt, &prow.DeletedAt,
			&addedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		items = append(items, m.Favorite{Product: prow.toModel(), AddedAt: addedAt})
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}

	const countSQL = `SELECT count(*) FROM favorites WHERE user_id = $1`
	if err = getConn(ctx, fr.pool).QueryRow(ctx, countSQL, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrToScan, err)
	}

	return items, total, nil
}

// Exists reports whether the user has favorited the product. addedAt is nil
// when favorited == false.
func (fr FavoriteRepoImpl) Exists(ctx context.Context, userID, productID int64) (favorited bool, addedAt *time.Time, err error) {
	const sql = `SELECT created_at FROM favorites WHERE user_id = $1 AND product_id = $2`
	var t time.Time
	if err = getConn(ctx, fr.pool).QueryRow(ctx, sql, userID, productID).Scan(&t); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return true, &t, nil
}
