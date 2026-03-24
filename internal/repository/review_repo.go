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

type ReviewRepoImpl struct {
	pool *pgxpool.Pool
}

func NewReviewRepo(pool *pgxpool.Pool) ReviewRepoImpl {
	return ReviewRepoImpl{pool: pool}
}

func (rr ReviewRepoImpl) GetReviewByID(ctx context.Context, id int64) (r m.Review, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "user_id", "product_id", "rating", "comment", "created_at").From("reviews").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return m.Review{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := rr.pool.QueryRow(ctx, sql, args...)
	var review reviewRow
	if err = row.Scan(&review.ID, &review.UserID, &review.ProductID, &review.Rating, &review.Comment, &review.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.Review{}, service.ErrNotFound
		}
		return m.Review{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return review.toModel(), nil
}

func (rr ReviewRepoImpl) GetReviewsByProductID(ctx context.Context, pid int64, opts m.PaginationOpts) (rs []m.Review, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sb := psql.Select("id", "user_id", "product_id", "rating", "comment", "created_at").From("reviews").Where(sq.Eq{"product_id": pid})
	sb = sb.Offset(uint64(opts.Page * opts.Limit)).Limit(uint64(opts.Limit))
	sql, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := rr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()
	rs = make([]m.Review, 0)
	var rrow reviewRow
	for rows.Next() {
		err = rows.Scan(&rrow.ID, &rrow.UserID, &rrow.ProductID, &rrow.Rating, &rrow.Comment, &rrow.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		rs = append(rs, rrow.toModel())
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return rs, nil
}

func (rr ReviewRepoImpl) CreateReview(ctx context.Context, rc m.ReviewCreate) (id int64, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Insert("reviews").Columns("user_id", "product_id", "rating", "comment").Values(rc.UserID, rc.ProductID, rc.Rating, rc.Comment).Suffix("RETURNING id").ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := rr.pool.QueryRow(ctx, sql, args...)
	if err = row.Scan(&id); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return id, nil
}

func (rr ReviewRepoImpl) UpdateReview(ctx context.Context, id int64, ru m.ReviewUpdate) (r m.Review, err error) {
	if ru.Rating == nil && ru.Comment == nil {
		return m.Review{}, service.ErrNoChangesInUpdate
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("reviews").Where(sq.Eq{"id": id})
	if ru.Rating != nil {
		ub = ub.Set("rating", *ru.Rating)
	}
	if ru.Comment != nil {
		ub = ub.Set("comment", *ru.Comment)
	}
	ub = ub.Suffix("RETURNING id, user_id, product_id, rating, comment, created_at")
	sql, args, err := ub.ToSql()
	if err != nil {
		return m.Review{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := rr.pool.QueryRow(ctx, sql, args...)
	var review reviewRow
	if err = row.Scan(&review.ID, &review.UserID, &review.ProductID, &review.Rating, &review.Comment, &review.CreatedAt); err != nil {
		return m.Review{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return review.toModel(), nil
}

func (rr ReviewRepoImpl) DeleteReviewByID(ctx context.Context, id int64) (err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Delete("reviews").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	if _, err = rr.pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}
