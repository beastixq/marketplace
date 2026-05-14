package repository

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"

	"github.com/jackc/pgerrcode"
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

func (fr FavoriteRepoImpl) AddFavorite(ctx context.Context, userID int64, productID int64) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Insert("favorites").
		Columns("user_id", "product_id").
		Values(userID, productID).
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	if _, err = getConn(ctx, fr.pool).Exec(ctx, sql, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return service.ErrFavoriteAlreadyExists
		}
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}

func (fr FavoriteRepoImpl) RemoveFavorite(ctx context.Context, userID int64, productID int64) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Delete("favorites").
		Where(sq.Eq{"user_id": userID, "product_id": productID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	tag, err := getConn(ctx, fr.pool).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	if tag.RowsAffected() == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (fr FavoriteRepoImpl) GetFavoritesByUserID(ctx context.Context, userID int64, opts m.PaginationOpts) ([]m.Favorite, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sb := psql.Select("user_id", "product_id", "created_at").
		From("favorites").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("created_at desc")
	if opts.Limit > 0 {
		sb = sb.Limit(uint64(opts.Limit)).Offset(uint64((opts.Page - 1) * opts.Limit))
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

	favorites := make([]m.Favorite, 0)
	var frow favoriteRow
	for rows.Next() {
		if err = rows.Scan(&frow.UserID, &frow.ProductID, &frow.CreatedAt); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		favorites = append(favorites, frow.toModel())
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return favorites, nil
}
