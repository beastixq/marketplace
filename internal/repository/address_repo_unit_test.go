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

func TestAddressRepo_GetAddressByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM addresses WHERE id = \\$1").WithArgs(int64(1)).WillReturnRows(addressRows("Moscow"))
		r := repository.NewAddressRepo(db)

		got, err := r.GetAddressByID(ctx, 1)

		requireErrorIs(t, err, nil)
		if got.ID != 1 || got.UserID != 2 {
			t.Fatalf("unexpected address: %+v", got)
		}
	})
	t.Run("query row failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM addresses WHERE id = \\$1").WithArgs(int64(9)).WillReturnError(unitErr)
		r := repository.NewAddressRepo(db)

		_, err := r.GetAddressByID(ctx, 9)

		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("not found", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM addresses WHERE id = \\$1").WithArgs(int64(9)).WillReturnError(pgx.ErrNoRows)
		r := repository.NewAddressRepo(db)

		_, err := r.GetAddressByID(ctx, 9)

		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestAddressRepo_GetAddressesByUserID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM addresses WHERE user_id = \\$1").WithArgs(int64(2)).WillReturnRows(addressRows("Moscow")).RowsWillBeClosed()
		r := repository.NewAddressRepo(db)

		got, err := r.GetAddressesByUserID(ctx, 2)

		requireErrorIs(t, err, nil)
		if len(got) != 1 || got[0].ID != 1 {
			t.Fatalf("unexpected addresses: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM addresses WHERE user_id = \\$1").WithArgs(int64(2)).WillReturnError(unitErr)
		r := repository.NewAddressRepo(db)

		_, err := r.GetAddressesByUserID(ctx, 2)

		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "user_id", "city", "street", "house", "zip_code", "is_default", "created_at"}).AddRow("invalid ID", int64(2), "Moscow", "Tverskaya", "1", "101000", true, unitNow), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: addressRows("Moscow").CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM addresses WHERE user_id = \\$1").WithArgs(int64(2)).WillReturnRows(tc.rows).RowsWillBeClosed()
			r := repository.NewAddressRepo(db)

			_, err := r.GetAddressesByUserID(ctx, 2)

			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestAddressRepo_CreateAddress(t *testing.T) {
	ctx := context.Background()

	create := m.AddressCreate{UserID: 2, City: "Moscow", Street: "Tverskaya", House: "1", ZipCode: "101000", IsDefault: true}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO addresses (.+) RETURNING id").WithArgs(create.UserID, create.City, create.Street, create.House, create.ZipCode, create.IsDefault).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))
		r := repository.NewAddressRepo(db)

		id, err := r.CreateAddress(ctx, create)

		requireErrorIs(t, err, nil)
		if id != 1 {
			t.Fatalf("id: got %d, want 1", id)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO addresses (.+) RETURNING id").WithArgs(create.UserID, create.City, create.Street, create.House, create.ZipCode, create.IsDefault).WillReturnError(unitErr)
		r := repository.NewAddressRepo(db)

		_, err := r.CreateAddress(ctx, create)

		requireErrorIs(t, err, repository.ErrToScan)
	})
}

func TestAddressRepo_UpdateAddress(t *testing.T) {
	ctx := context.Background()

	city := "Kazan"
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("UPDATE addresses SET city = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(city, int64(1)).WillReturnRows(addressRows(city))
		r := repository.NewAddressRepo(db)

		got, err := r.UpdateAddress(ctx, 1, m.AddressUpdate{City: &city})

		requireErrorIs(t, err, nil)
		if got.ID != 1 || got.City != city {
			t.Fatalf("unexpected address: %+v", got)
		}
	})
	t.Run("empty update", func(t *testing.T) {
		r := repository.NewAddressRepo(newMockPool(t))

		_, err := r.UpdateAddress(ctx, 1, m.AddressUpdate{})

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
			db.ExpectQuery("UPDATE addresses SET city = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(city, int64(1)).WillReturnError(tc.dbErr)
			r := repository.NewAddressRepo(db)
			_, err := r.UpdateAddress(ctx, 1, m.AddressUpdate{City: &city})
			requireErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestAddressRepo_DeleteAddressByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM addresses WHERE id = \\$1").WithArgs(int64(1)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		r := repository.NewAddressRepo(db)

		err := r.DeleteAddressByID(ctx, 1)

		requireErrorIs(t, err, nil)
	})
	t.Run("exec failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM addresses WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		r := repository.NewAddressRepo(db)

		err := r.DeleteAddressByID(ctx, 1)

		requireErrorIs(t, err, repository.ErrExec)
	})
}

func addressRows(city string) *pgxmock.Rows {
	addressColumns := []string{"id", "user_id", "city", "street", "house", "zip_code", "is_default", "created_at"}
	return pgxmock.NewRows(addressColumns).AddRow(int64(1), int64(2), city, "Tverskaya", "1", "101000", true, unitNow)
}
