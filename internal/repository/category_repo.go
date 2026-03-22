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

type CategoryRepoImpl struct {
	pool *pgxpool.Pool
}

func NewCategoryRepo(pool *pgxpool.Pool) CategoryRepoImpl {
	return CategoryRepoImpl{pool: pool}
}

func (cr CategoryRepoImpl) GetCategories(ctx context.Context, opts m.PaginationOpts) (cs []m.Category, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "parent_id", "name", "description").
		From("categories").
		Offset(uint64((opts.Page - 1) * opts.Limit)).
		Limit(uint64(opts.Limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := cr.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()
	cs = make([]m.Category, 0)
	var crow categoryRow
	for rows.Next() {
		if err = rows.Scan(&crow.ID, &crow.ParentID, &crow.Name, &crow.Description); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		cs = append(cs, crow.toModel())
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return cs, nil
}

func (cr CategoryRepoImpl) GetCategoryByID(ctx context.Context, id int64) (c m.Category, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "parent_id", "name", "description").
		From("categories").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return m.Category{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := cr.pool.QueryRow(ctx, sql, args...)
	var crow categoryRow
	if err = row.Scan(&crow.ID, &crow.ParentID, &crow.Name, &crow.Description); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.Category{}, service.ErrNotFound
		}
		return m.Category{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return crow.toModel(), nil
}

func (cr CategoryRepoImpl) CreateCategory(ctx context.Context, cc m.CategoryCreate) (id int64, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Insert("categories").
		Columns("parent_id", "name", "description").
		Values(cc.ParentID, cc.Name, cc.Description).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := cr.pool.QueryRow(ctx, sql, args...)
	if err = row.Scan(&id); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return id, nil
}

func (cr CategoryRepoImpl) UpdateCategory(ctx context.Context, id int64, cu m.CategoryUpdate) (c m.Category, err error) {
	if cu.ParentID == nil && cu.Name == nil && cu.Description == nil {
		return m.Category{}, service.ErrNoChangesInUpdate
	}
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("categories").Where(sq.Eq{"id": id})
	if cu.ParentID != nil {
		ub = ub.Set("parent_id", *cu.ParentID)
	}
	if cu.Name != nil {
		ub = ub.Set("name", *cu.Name)
	}
	if cu.Description != nil {
		ub = ub.Set("description", *cu.Description)
	}
	ub = ub.Suffix("RETURNING id, parent_id, name, description")
	sql, args, err := ub.ToSql()
	if err != nil {
		return m.Category{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := cr.pool.QueryRow(ctx, sql, args...)
	var crow categoryRow
	if err = row.Scan(&crow.ID, &crow.ParentID, &crow.Name, &crow.Description); err != nil {
		return m.Category{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return crow.toModel(), nil
}

func (cr CategoryRepoImpl) DeleteCategoryByID(ctx context.Context, id int64) (err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Delete("categories").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	if _, err = cr.pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}
