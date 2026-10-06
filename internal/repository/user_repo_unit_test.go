package repository_test

import (
	"context"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

func TestUserRepo_GetUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM users ORDER BY id ASC").WillReturnRows(userRows("buyer@example.test")).RowsWillBeClosed()
		r := repository.NewUserRepo(db)

		got, err := r.GetUsers(ctx, m.UserListOptions{})

		requireErrorIs(t, err, nil)
		if len(got) != 1 || got[0].ID != 1 {
			t.Fatalf("unexpected users: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM users ORDER BY id ASC").WillReturnError(unitErr)
		r := repository.NewUserRepo(db)

		_, err := r.GetUsers(ctx, m.UserListOptions{})

		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "email", "password_hash", "full_name", "phone", "role", "created_at", "deleted_at"}).AddRow("invalid ID", "buyer@example.test", "hash", "Buyer", nil, m.RoleBuyer, unitNow, nil), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: userRows("buyer@example.test").CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM users ORDER BY id ASC").WillReturnRows(tc.rows).RowsWillBeClosed()
			r := repository.NewUserRepo(db)

			_, err := r.GetUsers(ctx, m.UserListOptions{})

			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestUserRepo_GetUserByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM users WHERE id = \\$1").WithArgs(int64(1)).WillReturnRows(userRows("buyer@example.test"))
		r := repository.NewUserRepo(db)

		got, err := r.GetUserByID(ctx, 1)

		requireErrorIs(t, err, nil)
		if got.ID != 1 {
			t.Fatalf("unexpected user: %+v", got)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM users WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		r := repository.NewUserRepo(db)

		_, err := r.GetUserByID(ctx, 1)

		requireErrorIs(t, err, repository.ErrToScan)
	})
}

func TestUserRepo_GetUserByEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM users WHERE email = \\$1").WithArgs("buyer@example.test").WillReturnRows(userRows("buyer@example.test"))
		r := repository.NewUserRepo(db)

		got, err := r.GetUserByEmail(ctx, "buyer@example.test")

		requireErrorIs(t, err, nil)
		if got.ID != 1 {
			t.Fatalf("unexpected user: %+v", got)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM users WHERE email = \\$1").WithArgs("buyer@example.test").WillReturnError(unitErr)
		r := repository.NewUserRepo(db)

		_, err := r.GetUserByEmail(ctx, "buyer@example.test")

		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("not found", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM users WHERE email = \\$1").WithArgs("buyer@example.test").WillReturnError(pgx.ErrNoRows)
		r := repository.NewUserRepo(db)

		_, err := r.GetUserByEmail(ctx, "buyer@example.test")

		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestUserRepo_CreateUser(t *testing.T) {
	ctx := context.Background()

	create := m.UserCreate{Email: "buyer@example.test", Password: "hash", FullName: "Buyer", Role: m.RoleBuyer}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO users (.+) RETURNING id").WithArgs(create.Email, create.Password, create.FullName, pgxmock.AnyArg(), create.Role).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))
		r := repository.NewUserRepo(db)

		id, err := r.CreateUser(ctx, create)

		requireErrorIs(t, err, nil)
		if id != 1 {
			t.Fatalf("id: got %d, want 1", id)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO users (.+) RETURNING id").WithArgs(create.Email, create.Password, create.FullName, pgxmock.AnyArg(), create.Role).WillReturnError(unitErr)
		r := repository.NewUserRepo(db)

		_, err := r.CreateUser(ctx, create)

		requireErrorIs(t, err, repository.ErrToScan)
	})
}

func TestUserRepo_UpdateUser(t *testing.T) {
	ctx := context.Background()

	email := "new@example.test"
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("UPDATE users SET email = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(email, int64(1)).WillReturnRows(userRows(email))
		r := repository.NewUserRepo(db)

		got, err := r.UpdateUser(ctx, 1, m.UserUpdate{Email: &email})

		requireErrorIs(t, err, nil)
		if got.ID != 1 || got.Email != email {
			t.Fatalf("unexpected user: %+v", got)
		}
	})
	t.Run("empty update", func(t *testing.T) {
		r := repository.NewUserRepo(newMockPool(t))

		_, err := r.UpdateUser(ctx, 1, m.UserUpdate{})

		requireErrorIs(t, err, service.ErrNoChangesInUpdate)
	})
	for _, tc := range []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{name: "query row failure", dbErr: unitErr, wantErr: repository.ErrToScan},
		{name: "not found", dbErr: pgx.ErrNoRows, wantErr: service.ErrNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newMockPool(t)
			db.ExpectQuery("UPDATE users SET email = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(email, int64(1)).WillReturnError(tc.dbErr)
			r := repository.NewUserRepo(db)
			_, err := r.UpdateUser(ctx, 1, m.UserUpdate{Email: &email})
			requireErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestUserRepo_ChangePasswordUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("UPDATE users SET password_hash = \\$1 WHERE id = \\$2").WithArgs("new-hash", int64(1)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		r := repository.NewUserRepo(db)

		err := r.ChangePasswordUser(ctx, 1, "new-hash")

		requireErrorIs(t, err, nil)
	})
	t.Run("exec failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("UPDATE users SET password_hash = \\$1 WHERE id = \\$2").WithArgs("new-hash", int64(1)).WillReturnError(unitErr)
		r := repository.NewUserRepo(db)

		err := r.ChangePasswordUser(ctx, 1, "new-hash")

		requireErrorIs(t, err, repository.ErrExec)
	})
}

func TestUserRepo_DeleteUserByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("UPDATE users SET deleted_at = NOW\\(\\) WHERE id = \\$1").WithArgs(int64(1)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		r := repository.NewUserRepo(db)

		err := r.DeleteUserByID(ctx, 1)

		requireErrorIs(t, err, nil)
	})
	t.Run("exec failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("UPDATE users SET deleted_at = NOW\\(\\) WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		r := repository.NewUserRepo(db)

		err := r.DeleteUserByID(ctx, 1)

		requireErrorIs(t, err, repository.ErrExec)
	})
}

func userRows(email string) *pgxmock.Rows {
	columns := []string{"id", "email", "password_hash", "full_name", "phone", "role", "created_at", "deleted_at"}
	return pgxmock.NewRows(columns).AddRow(int64(1), email, "hash", "Buyer", nil, m.RoleBuyer, unitNow, nil)
}
