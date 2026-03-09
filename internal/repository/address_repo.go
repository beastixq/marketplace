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

type AddressRepoImpl struct {
	pool *pgxpool.Pool
}

func NewAddressRepo(pool *pgxpool.Pool) AddressRepoImpl {
	return AddressRepoImpl{pool: pool}
}

func (ar AddressRepoImpl) GetAddressByID(ctx context.Context, id int64) (a m.Address, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "user_id", "city", "street", "zip_code", "is_default", "created_at").From("addresses").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return m.Address{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := ar.pool.QueryRow(ctx, sql, args...)
	var address addressRow
	if err = row.Scan(&address.ID, &address.UserID, &address.City, &address.Street, &address.ZipCode, &address.IsDefault, &address.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.Address{}, service.ErrNotFound
		}
		return m.Address{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return address.toModel(), nil
}

func (ar AddressRepoImpl) GetAddressesByUserID(ctx context.Context, userID int64) (ads []m.Address, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "user_id", "city", "street", "zip_code", "is_default", "created_at").From("addresses").Where(sq.Eq{"user_id": userID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := ar.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()
	addresses := make([]addressRow, 0)
	var addr addressRow
	for rows.Next() {
		err = rows.Scan(&addr.ID, &addr.UserID, &addr.City, &addr.Street, &addr.ZipCode, &addr.IsDefault, &addr.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		addresses = append(addresses, addr)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	ads = make([]m.Address, len(addresses))
	for i, a := range addresses {
		ads[i] = a.toModel()
	}
	return ads, nil
}

func (ar AddressRepoImpl) CreateAddress(ctx context.Context, ac m.AddressCreate) (id int64, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Insert("addresses").Columns("user_id", "city", "street", "zip_code", "is_default").Values(ac.UserID, ac.City, ac.Street, ac.ZipCode, ac.IsDefault).Suffix("RETURNING id").ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := ar.pool.QueryRow(ctx, sql, args...)
	if err = row.Scan(&id); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return id, nil
}

func (ar AddressRepoImpl) UpdateAddress(ctx context.Context, id int64, au m.AddressUpdate) (a m.Address, err error) {
	if au.UserID == nil && au.City == nil && au.Street == nil && au.ZipCode == nil && au.IsDefault == nil {
		return m.Address{}, service.ErrNoChangesInUpdate
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("addresses").Where(sq.Eq{"id": id})
	if au.UserID != nil {
		ub = ub.Set("user_id", *au.UserID)
	}
	if au.City != nil {
		ub = ub.Set("city", *au.City)
	}
	if au.Street != nil {
		ub = ub.Set("street", *au.Street)
	}
	if au.ZipCode != nil {
		ub = ub.Set("zip_code", *au.ZipCode)
	}
	if au.IsDefault != nil {
		ub = ub.Set("is_default", *au.IsDefault)
	}
	ub = ub.Suffix("RETURNING id, user_id, city, street, zip_code, is_default, created_at")
	sql, args, err := ub.ToSql()
	if err != nil {
		return m.Address{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := ar.pool.QueryRow(ctx, sql, args...)
	var address addressRow
	if err = row.Scan(&address.ID, &address.UserID, &address.City, &address.Street, &address.ZipCode, &address.IsDefault, &address.CreatedAt); err != nil {
		return m.Address{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return address.toModel(), nil
}

func (ar AddressRepoImpl) DeleteAddressByID(ctx context.Context, id int64) (err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Delete("addresses").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	if _, err = ar.pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}
