package repository

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepoImpl struct {
	pool *pgxpool.Pool
}

func GetNewUserRepo(pool *pgxpool.Pool) UserRepoImpl {
	return UserRepoImpl{pool: pool}
}

func (ur UserRepoImpl) GetUserByID(ctx context.Context, id int64) (u m.User, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("users").Where("id = ", id).ToSql()
	if err != nil {
		return m.User{}, fmt.Errorf("GetUserByID: %v", ErrToSql)
	}
	row := ur.pool.QueryRow(ctx, sql, args...)
	var user userRow
	if err = row.Scan(&user); err != nil {
		return m.User{}, fmt.Errorf("GetUserByID: %v", ErrToScan)
	}
	return user.toModel(), nil
}

// GetUserByEmail(ctx context.Context, email string) (u m.User, err error) {

// }
// CreateUser(ctx context.Context, uc m.UserCreate) (id int64, err error) {

// }
// UpdateUser(ctx context.Context, id int64, uu m.UserUpdate) (u m.User, err error) {

// }
// ChangePasswordUser(ctx context.Context, id int64, newPassHash string) (err error) {

// }
// DeleteUserByID(ctx context.Context, id int64) (err error) {

// }
