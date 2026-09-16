package repository_test

import (
	"context"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/shopspring/decimal"
)

// Тесты триггеров БД (см. РПЗ, раздел «Реализация триггеров и хранимой функции»):
//
//	catch_price_change       (миграция 004) — журналирование изменения цены товара;
//	update_ratings_on_review (миграция 007) — пересчёт рейтинга товара и продавца.

// updatePriceAs изменяет цену товара в отдельной транзакции, предварительно
// задавая app.current_user: его читает функция триггера через current_setting
// и пишет в столбец changed_by.
func updatePriceAs(t *testing.T, productID int64, newPrice float64, actor string) {
	t.Helper()
	ctx := context.Background()
	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("updatePriceAs: begin: %v", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_user', $1, true)", actor); err != nil {
		t.Fatalf("updatePriceAs: set_config: %v", err)
	}
	if _, err := tx.Exec(ctx, "UPDATE products SET price = $1 WHERE id = $2", newPrice, productID); err != nil {
		t.Fatalf("updatePriceAs: update: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("updatePriceAs: commit: %v", err)
	}
}

func historyCount(t *testing.T, productID int64) int {
	t.Helper()
	var n int
	if err := testPool.QueryRow(context.Background(),
		"SELECT count(*) FROM product_price_history WHERE product_id = $1", productID).Scan(&n); err != nil {
		t.Fatalf("historyCount: %v", err)
	}
	return n
}

// Изменение цены записывает строку истории с прежней/новой ценой и актором.
func TestTrigger_PriceChange_RecordsHistory(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := statsProduct(t, sellerID, "price trigger product") // стартовая цена 100
	ctx := context.Background()

	updatePriceAs(t, productID, 150, "auditor@example.com")

	if n := historyCount(t, productID); n != 1 {
		t.Fatalf("ожидалась 1 запись истории, получено %d", n)
	}

	var oldP, newP decimal.Decimal
	var changedBy string
	if err := testPool.QueryRow(ctx,
		"SELECT old_price, new_price, changed_by FROM product_price_history WHERE product_id = $1 ORDER BY id DESC LIMIT 1",
		productID).Scan(&oldP, &newP, &changedBy); err != nil {
		t.Fatalf("read history: %v", err)
	}
	if !oldP.Equal(decimal.NewFromInt(100)) {
		t.Errorf("old_price = %s, want 100", oldP)
	}
	if !newP.Equal(decimal.NewFromInt(150)) {
		t.Errorf("new_price = %s, want 150", newP)
	}
	if changedBy != "auditor@example.com" {
		t.Errorf("changed_by = %q, want %q", changedBy, "auditor@example.com")
	}
}

// Условие WHEN (OLD.price IS DISTINCT FROM NEW.price) исключает запись истории,
// когда цена не изменилась.
func TestTrigger_PriceChange_SamePriceNoHistory(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := statsProduct(t, sellerID, "same price product") // цена 100

	updatePriceAs(t, productID, 100, "auditor@example.com")

	if n := historyCount(t, productID); n != 0 {
		t.Errorf("при неизменной цене ожидалось 0 записей истории, получено %d", n)
	}
}

// changed_by заполняется значением конфигурационного параметра сессии
// app.current_user (custom GUC), который функция триггера читает через
// current_setting. Разные значения параметра дают разный changed_by.
func TestTrigger_PriceChange_ChangedByFromGUC(t *testing.T) {
	actors := []string{"seller:42", "admin:1", "support:7:techui"}
	for _, actor := range actors {
		t.Run(actor, func(t *testing.T) {
			sellerID := createTestSeller(t) 
			productID := statsProduct(t, sellerID, "guc product") // цена 100

			// updatePriceAs задаёт app.current_user = actor и меняет цену
			// в одной транзакции, поэтому триггер видит именно это значение.
			updatePriceAs(t, productID, 250, actor)

			var changedBy string
			if err := testPool.QueryRow(context.Background(),
				"SELECT changed_by FROM product_price_history WHERE product_id = $1 ORDER BY id DESC LIMIT 1",
				productID).Scan(&changedBy); err != nil {
				t.Fatalf("read changed_by: %v", err)
			}
			if changedBy != actor {
				t.Errorf("changed_by = %q, want %q (значение из app.current_user)", changedBy, actor)
			}
		})
	}
}

// ratingOf читает столбец rating (numeric(3,2)) указанной таблицы.
// table — внутренняя константа теста, не пользовательский ввод.
func ratingOf(t *testing.T, table string, id int64) decimal.NullDecimal {
	t.Helper()
	var nd decimal.NullDecimal
	if err := testPool.QueryRow(context.Background(),
		"SELECT rating FROM "+table+" WHERE id = $1", id).Scan(&nd); err != nil {
		t.Fatalf("read %s.rating: %v", table, err)
	}
	return nd
}

func assertRating(t *testing.T, label string, got decimal.NullDecimal, want float64) {
	t.Helper()
	if !got.Valid {
		t.Fatalf("%s: rating = NULL, want %v", label, want)
	}
	if !got.Decimal.Equal(decimal.NewFromFloat(want)) {
		t.Errorf("%s: rating = %s, want %v", label, got.Decimal, want)
	}
}

func assertRatingNull(t *testing.T, label string, got decimal.NullDecimal) {
	t.Helper()
	if got.Valid {
		t.Errorf("%s: rating = %s, want NULL", label, got.Decimal)
	}
}

// addReview создаёт отзыв от нового покупателя (UNIQUE(user_id, product_id)).
func addReview(t *testing.T, productID int64, rating int8) (reviewID int64) {
	t.Helper()
	buyerID := createTestUser(t)
	r := repo.NewReviewRepo(testPool)
	id, err := r.CreateReview(context.Background(), m.ReviewCreate{
		UserID:    buyerID,
		ProductID: productID,
		Rating:    rating,
	})
	if err != nil {
		t.Fatalf("addReview(rating=%d): %v", rating, err)
	}
	return id
}

// Добавление отзыва задаёт рейтинг товара и рейтинг продавца.
func TestTrigger_Rating_InsertSetsProductAndSeller(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := statsProduct(t, sellerID, "rating insert product")

	assertRatingNull(t, "product до отзыва", ratingOf(t, "products", productID))
	assertRatingNull(t, "seller до отзыва", ratingOf(t, "sellers", sellerID))

	addReview(t, productID, 4)

	assertRating(t, "product", ratingOf(t, "products", productID), 4)
	assertRating(t, "seller", ratingOf(t, "sellers", sellerID), 4)
}

// Рейтинг товара — среднее по всем его отзывам.
func TestTrigger_Rating_AverageOfReviews(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := statsProduct(t, sellerID, "rating avg product")

	addReview(t, productID, 4)
	addReview(t, productID, 2)

	assertRating(t, "product", ratingOf(t, "products", productID), 3)
	assertRating(t, "seller", ratingOf(t, "sellers", sellerID), 3)
}

// Изменение оценки отзыва пересчитывает рейтинг (UPDATE OF rating).
func TestTrigger_Rating_UpdateRecalculates(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := statsProduct(t, sellerID, "rating update product")
	ctx := context.Background()

	rev1 := addReview(t, productID, 4)
	addReview(t, productID, 2)
	assertRating(t, "product до правки", ratingOf(t, "products", productID), 3)

	newRating := int8(5)
	if _, err := repo.NewReviewRepo(testPool).UpdateReview(ctx, rev1, m.ReviewUpdate{Rating: &newRating}); err != nil {
		t.Fatalf("UpdateReview: %v", err)
	}

	assertRating(t, "product после правки", ratingOf(t, "products", productID), 3.5)
}

// Удаление отзыва пересчитывает рейтинг; удаление последнего отзыва обнуляет
// рейтинг товара (AVG по пустому множеству = NULL) и рейтинг продавца.
func TestTrigger_Rating_DeleteRecalculates(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := statsProduct(t, sellerID, "rating delete product")
	ctx := context.Background()
	rr := repo.NewReviewRepo(testPool)

	rev1 := addReview(t, productID, 4)
	rev2 := addReview(t, productID, 2)
	assertRating(t, "product до удаления", ratingOf(t, "products", productID), 3)

	if err := rr.DeleteReviewByID(ctx, rev2); err != nil {
		t.Fatalf("DeleteReviewByID(rev2): %v", err)
	}
	assertRating(t, "product после удаления одного", ratingOf(t, "products", productID), 4)

	if err := rr.DeleteReviewByID(ctx, rev1); err != nil {
		t.Fatalf("DeleteReviewByID(rev1): %v", err)
	}
	assertRatingNull(t, "product без отзывов", ratingOf(t, "products", productID))
	assertRatingNull(t, "seller без оценённых товаров", ratingOf(t, "sellers", sellerID))
}

// Рейтинг продавца — среднее по рейтингам его оценённых товаров.
func TestTrigger_Rating_SellerAcrossProducts(t *testing.T) {
	sellerID := createTestSeller(t)
	p1 := statsProduct(t, sellerID, "seller rating p1")
	p2 := statsProduct(t, sellerID, "seller rating p2")

	addReview(t, p1, 5)
	addReview(t, p2, 3)

	assertRating(t, "product p1", ratingOf(t, "products", p1), 5)
	assertRating(t, "product p2", ratingOf(t, "products", p2), 3)
	assertRating(t, "seller", ratingOf(t, "sellers", sellerID), 4)
}
