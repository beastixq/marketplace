package repository_test

import (
	"context"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"
)

func TestReviewRepo_GetReviewByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM reviews WHERE id = \\$1").WithArgs(int64(1)).WillReturnRows(reviewRows(int8(5)))
		got, err := repository.NewReviewRepo(db).GetReviewByID(ctx, 1)
		requireErrorIs(t, err, nil)
		if got.ID != 1 {
			t.Fatalf("unexpected review: %+v", got)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM reviews WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		_, err := repository.NewReviewRepo(db).GetReviewByID(ctx, 1)
		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("not found", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM reviews WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(pgx.ErrNoRows)
		_, err := repository.NewReviewRepo(db).GetReviewByID(ctx, 1)
		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestReviewRepo_GetReviewsByProductID(t *testing.T) {
	ctx := context.Background()

	opts := m.PaginationOpts{Page: 1, Limit: 10}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM reviews WHERE product_id = \\$1 LIMIT 10 OFFSET 0").WithArgs(int64(3)).WillReturnRows(reviewRows(int8(5))).RowsWillBeClosed()
		got, err := repository.NewReviewRepo(db).GetReviewsByProductID(ctx, 3, opts)
		requireErrorIs(t, err, nil)
		if len(got) != 1 {
			t.Fatalf("unexpected reviews: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM reviews WHERE product_id = \\$1 LIMIT 10 OFFSET 0").WithArgs(int64(3)).WillReturnError(unitErr)
		_, err := repository.NewReviewRepo(db).GetReviewsByProductID(ctx, 3, opts)
		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "user_id", "product_id", "rating", "comment", "created_at"}).AddRow("invalid ID", int64(2), int64(3), int8(5), nil, unitNow), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: reviewRows(int8(5)).CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM reviews WHERE product_id = \\$1 LIMIT 10 OFFSET 0").WithArgs(int64(3)).WillReturnRows(tc.rows).RowsWillBeClosed()
			_, err := repository.NewReviewRepo(db).GetReviewsByProductID(ctx, 3, opts)
			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestReviewRepo_UserPurchasedProduct(t *testing.T) {
	ctx := context.Background()

	t.Run("purchased", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("(?s)SELECT EXISTS (.+)").WithArgs(int64(2), int64(3)).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
		got, err := repository.NewReviewRepo(db).UserPurchasedProduct(ctx, 2, 3)
		requireErrorIs(t, err, nil)
		if !got {
			t.Fatal("expected purchase")
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("(?s)SELECT EXISTS (.+)").WithArgs(int64(2), int64(3)).WillReturnError(unitErr)
		_, err := repository.NewReviewRepo(db).UserPurchasedProduct(ctx, 2, 3)
		requireErrorIs(t, err, repository.ErrToScan)
	})
}

func TestReviewRepo_CreateReview(t *testing.T) {
	ctx := context.Background()

	create := m.ReviewCreate{UserID: 2, ProductID: 3, Rating: 5}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO reviews (.+) RETURNING id").WithArgs(create.UserID, create.ProductID, create.Rating, pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))
		id, err := repository.NewReviewRepo(db).CreateReview(ctx, create)
		requireErrorIs(t, err, nil)
		if id != 1 {
			t.Fatalf("id: got %d, want 1", id)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO reviews (.+) RETURNING id").WithArgs(create.UserID, create.ProductID, create.Rating, pgxmock.AnyArg()).WillReturnError(unitErr)
		_, err := repository.NewReviewRepo(db).CreateReview(ctx, create)
		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("duplicate review", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO reviews (.+) RETURNING id").WithArgs(create.UserID, create.ProductID, create.Rating, pgxmock.AnyArg()).WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "reviews_user_id_product_id_key"})
		r := repository.NewReviewRepo(db)

		_, err := r.CreateReview(ctx, create)

		requireErrorIs(t, err, service.ErrReviewAlreadyExists)
	})
}

func TestReviewRepo_UpdateReview(t *testing.T) {
	ctx := context.Background()

	rating := int8(4)
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("UPDATE reviews SET rating = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(rating, int64(1)).WillReturnRows(reviewRows(rating))
		got, err := repository.NewReviewRepo(db).UpdateReview(ctx, 1, m.ReviewUpdate{Rating: &rating})
		requireErrorIs(t, err, nil)
		if got.ID != 1 || got.Rating != rating {
			t.Fatalf("unexpected review: %+v", got)
		}
	})
	t.Run("empty update", func(t *testing.T) {
		_, err := repository.NewReviewRepo(newMockPool(t)).UpdateReview(ctx, 1, m.ReviewUpdate{})
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
			db.ExpectQuery("UPDATE reviews SET rating = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(rating, int64(1)).WillReturnError(tc.dbErr)
			r := repository.NewReviewRepo(db)
			_, err := r.UpdateReview(ctx, 1, m.ReviewUpdate{Rating: &rating})
			requireErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestReviewRepo_DeleteReviewByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM reviews WHERE id = \\$1").WithArgs(int64(1)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		err := repository.NewReviewRepo(db).DeleteReviewByID(ctx, 1)
		requireErrorIs(t, err, nil)
	})
	t.Run("exec failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM reviews WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		err := repository.NewReviewRepo(db).DeleteReviewByID(ctx, 1)
		requireErrorIs(t, err, repository.ErrExec)
	})
}

func reviewRows(rating int8) *pgxmock.Rows {
	columns := []string{"id", "user_id", "product_id", "rating", "comment", "created_at"}
	return pgxmock.NewRows(columns).AddRow(int64(1), int64(2), int64(3), rating, nil, unitNow)
}
