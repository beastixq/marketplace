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

type UserRepoImpl struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) UserRepoImpl {
	return UserRepoImpl{pool: pool}
}

func (ur UserRepoImpl) GetUserByID(ctx context.Context, id int64) (u m.User, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.
		Select("id", "email", "password_hash", "full_name", "phone", "role", "created_at", "deleted_at").
		From("users").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return m.User{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := getConn(ctx, ur.pool).QueryRow(ctx, sql, args...)
	var user userRow
	if err = row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Phone, &user.Role, &user.CreatedAt, &user.DeletedAt); err != nil {
		return m.User{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return user.toModel(), nil
}

func (ur UserRepoImpl) GetUserByEmail(ctx context.Context, email string) (u m.User, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "email", "password_hash", "full_name", "phone", "role", "created_at", "deleted_at").From("users").Where(sq.Eq{"email": email}).ToSql()
	if err != nil {
		return m.User{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := getConn(ctx, ur.pool).QueryRow(ctx, sql, args...)
	var user userRow
	if err = row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Phone, &user.Role, &user.CreatedAt, &user.DeletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.User{}, service.ErrNotFound
		}
		return m.User{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return user.toModel(), nil
}

func (ur UserRepoImpl) CreateUser(ctx context.Context, uc m.UserCreate) (id int64, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.
		Insert("users").
		Columns("email", "password_hash", "full_name", "phone", "role").
		Values(uc.Email, uc.Password, uc.FullName, uc.Phone, uc.Role).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := getConn(ctx, ur.pool).QueryRow(ctx, sql, args...)
	if err = row.Scan(&id); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return id, nil
}

func (ur UserRepoImpl) UpdateUser(ctx context.Context, id int64, uu m.UserUpdate) (u m.User, err error) {
	if uu.Email == nil && uu.FullName == nil && uu.Phone == nil && uu.Role == nil {
		return m.User{}, service.ErrNoChangesInUpdate
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("users").Where(sq.Eq{"id": id})
	if uu.Email != nil {
		ub = ub.Set("email", *uu.Email)
	}
	if uu.FullName != nil {
		ub = ub.Set("full_name", *uu.FullName)
	}
	if uu.Phone != nil {
		ub = ub.Set("phone", *uu.Phone)
	}
	if uu.Role != nil {
		ub = ub.Set("role", *uu.Role)
	}
	ub = ub.Suffix("RETURNING id, email, password_hash, full_name, phone, role, created_at, deleted_at")
	sql, args, err := ub.ToSql()
	if err != nil {
		return m.User{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := getConn(ctx, ur.pool).QueryRow(ctx, sql, args...)
	var user userRow
	if err = row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Phone, &user.Role, &user.CreatedAt, &user.DeletedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.User{}, service.ErrNotFound
		}
		return m.User{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return user.toModel(), nil
}

func (ur UserRepoImpl) ChangePasswordUser(ctx context.Context, id int64, newPassHash string) (err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Update("users").Where(sq.Eq{"id": id}).Set("password_hash", newPassHash).ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	if _, err = getConn(ctx, ur.pool).Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}

func (ur UserRepoImpl) DeleteUserByID(ctx context.Context, id int64) (err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Update("users").Where(sq.Eq{"id": id}).Set("deleted_at", sq.Expr("NOW()")).ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	if _, err = getConn(ctx, ur.pool).Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}
