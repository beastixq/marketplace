package repository_test

import (
	"context"
	"errors"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
)

var _ service.ReviewRepo = repo.ReviewRepoImpl{}

// createTestProduct создаёт товар для переданного продавца и регистрирует очистку.
func createTestProduct(t *testing.T, sellerID int64) int64 {
	t.Helper()
	ctx := context.Background()
	r := repo.NewProductRepo(testPool)
	id, err := r.CreateProduct(ctx, m.ProductCreate{
		SellerID:      sellerID,
		Name:          "Review Test Product",
		Price:         decimal.NewFromFloat(100),
		StockQuantity: 1,
	})
	if err != nil {
		t.Fatalf("createTestProduct: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM product_price_history WHERE product_id = $1", id)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", id)
	})
	return id
}

func TestReviewRepo_CreateAndGet(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	buyerID := createTestUser(t)
	var r service.ReviewRepo = repo.NewReviewRepo(testPool)
	ctx := context.Background()

	comment := "Great product"
	id, err := r.CreateReview(ctx, m.ReviewCreate{
		UserID:    buyerID,
		ProductID: productID,
		Rating:    5,
		Comment:   &comment,
	})
	if err != nil {
		t.Fatalf("CreateReview: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM reviews WHERE id = $1", id)
	})

	got, err := r.GetReviewByID(ctx, id)
	if err != nil {
		t.Fatalf("GetReviewByID: %v", err)
	}
	if got.UserID != buyerID {
		t.Errorf("UserID: got %d, want %d", got.UserID, buyerID)
	}
	if got.ProductID != productID {
		t.Errorf("ProductID: got %d, want %d", got.ProductID, productID)
	}
	if got.Rating != 5 {
		t.Errorf("Rating: got %d, want 5", got.Rating)
	}
}

func TestReviewRepo_GetByProductID(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	buyer1 := createTestUser(t)
	buyer2 := createTestUser(t)
	var r service.ReviewRepo = repo.NewReviewRepo(testPool)
	ctx := context.Background()

	// Pre-condition: отзывов на товар нет.
	before, err := r.GetReviewsByProductID(ctx, productID, m.PaginationOpts{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetReviewsByProductID до создания: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("ожидается 0 отзывов, получено %d", len(before))
	}

	id1, err := r.CreateReview(ctx, m.ReviewCreate{UserID: buyer1, ProductID: productID, Rating: 4})
	if err != nil {
		t.Fatalf("CreateReview 1: %v", err)
	}
	id2, err := r.CreateReview(ctx, m.ReviewCreate{UserID: buyer2, ProductID: productID, Rating: 5})
	if err != nil {
		t.Fatalf("CreateReview 2: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM reviews WHERE id = $1", id1)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM reviews WHERE id = $1", id2)
	})

	after, err := r.GetReviewsByProductID(ctx, productID, m.PaginationOpts{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetReviewsByProductID: %v", err)
	}
	if len(after) != 2 {
		t.Errorf("ожидается 2 отзыва, получено %d", len(after))
	}
	ratings := map[int8]bool{}
	for _, rv := range after {
		if rv.ProductID != productID {
			t.Errorf("отзыв с чужим ProductID=%d", rv.ProductID)
		}
		ratings[rv.Rating] = true
	}
	if !ratings[4] || !ratings[5] {
		t.Errorf("ожидаются рейтинги 4 и 5, получено: %v", ratings)
	}
}

func TestReviewRepo_Update(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	buyerID := createTestUser(t)
	var r service.ReviewRepo = repo.NewReviewRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateReview(ctx, m.ReviewCreate{
		UserID: buyerID, ProductID: productID, Rating: 3,
	})
	if err != nil {
		t.Fatalf("CreateReview: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM reviews WHERE id = $1", id)
	})

	newRating := int8(5)
	newComment := "Changed my mind"
	updated, err := r.UpdateReview(ctx, id, m.ReviewUpdate{
		Rating:  &newRating,
		Comment: &newComment,
	})
	if err != nil {
		t.Fatalf("UpdateReview: %v", err)
	}
	if updated.Rating != newRating {
		t.Errorf("Rating: got %d, want %d", updated.Rating, newRating)
	}
	if updated.Comment == nil || *updated.Comment != newComment {
		t.Errorf("Comment: got %v, want %q", updated.Comment, newComment)
	}

	fetched, err := r.GetReviewByID(ctx, id)
	if err != nil {
		t.Fatalf("GetReviewByID после Update: %v", err)
	}
	if fetched.Rating != newRating {
		t.Errorf("Rating в БД: got %d, want %d", fetched.Rating, newRating)
	}
	if fetched.Comment == nil || *fetched.Comment != newComment {
		t.Errorf("Comment в БД: got %v, want %q", fetched.Comment, newComment)
	}
}

func TestReviewRepo_Create_Duplicate(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	buyerID := createTestUser(t)
	var r service.ReviewRepo = repo.NewReviewRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateReview(ctx, m.ReviewCreate{UserID: buyerID, ProductID: productID, Rating: 4})
	if err != nil {
		t.Fatalf("CreateReview первый: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM reviews WHERE id = $1", id)
	})

	_, err = r.CreateReview(ctx, m.ReviewCreate{UserID: buyerID, ProductID: productID, Rating: 5})
	if err == nil {
		t.Error("второй отзыв от того же пользователя на тот же товар должен вернуть ошибку")
	}
}

func TestReviewRepo_UserPurchasedProduct(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	buyerID := createTestUser(t)
	otherBuyerID := createTestUser(t)
	ctx := context.Background()
	reviews := repo.NewReviewRepo(testPool)
	orders := repo.NewOrderRepo(testPool)
	items := repo.NewOrderItemRepo(testPool)

	paidOrderID, err := orders.CreateOrder(ctx, m.OrderCreate{
		UserID:      buyerID,
		SellerID:    &sellerID,
		Status:      m.StatusPaid,
		TotalAmount: decimal.NewFromFloat(100),
	})
	if err != nil {
		t.Fatalf("CreateOrder paid: %v", err)
	}
	itemID, err := items.CreateOrderItem(ctx, m.OrderItemCreate{
		OrderID:         paidOrderID,
		ProductID:       productID,
		Quantity:        1,
		PriceAtPurchase: decimal.NewFromFloat(100),
	})
	if err != nil {
		t.Fatalf("CreateOrderItem paid: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM order_items WHERE id = $1", itemID)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", paidOrderID)
	})

	purchased, err := reviews.UserPurchasedProduct(ctx, buyerID, productID)
	if err != nil {
		t.Fatalf("UserPurchasedProduct paid: %v", err)
	}
	if !purchased {
		t.Fatal("expected paid order to allow review")
	}

	purchased, err = reviews.UserPurchasedProduct(ctx, otherBuyerID, productID)
	if err != nil {
		t.Fatalf("UserPurchasedProduct other buyer: %v", err)
	}
	if purchased {
		t.Fatal("other buyer must not be treated as purchaser")
	}

	draftOrderID, err := orders.CreateOrder(ctx, m.OrderCreate{
		UserID:      otherBuyerID,
		SellerID:    &sellerID,
		Status:      m.StatusDraft,
		TotalAmount: decimal.Zero,
	})
	if err != nil {
		t.Fatalf("CreateOrder draft: %v", err)
	}
	draftItemID, err := items.CreateOrderItem(ctx, m.OrderItemCreate{
		OrderID:         draftOrderID,
		ProductID:       productID,
		Quantity:        1,
		PriceAtPurchase: decimal.NewFromFloat(100),
	})
	if err != nil {
		t.Fatalf("CreateOrderItem draft: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM order_items WHERE id = $1", draftItemID)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", draftOrderID)
	})

	purchased, err = reviews.UserPurchasedProduct(ctx, otherBuyerID, productID)
	if err != nil {
		t.Fatalf("UserPurchasedProduct draft: %v", err)
	}
	if purchased {
		t.Fatal("draft order must not allow review")
	}
}

func TestReviewRepo_Update_NotFound(t *testing.T) {
	var r service.ReviewRepo = repo.NewReviewRepo(testPool)
	ctx := context.Background()

	rating := int8(5)
	_, err := r.UpdateReview(ctx, 999999999, m.ReviewUpdate{Rating: &rating})
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestReviewRepo_Delete(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	buyerID := createTestUser(t)
	var r service.ReviewRepo = repo.NewReviewRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateReview(ctx, m.ReviewCreate{
		UserID: buyerID, ProductID: productID, Rating: 3,
	})
	if err != nil {
		t.Fatalf("CreateReview: %v", err)
	}

	if err := r.DeleteReviewByID(ctx, id); err != nil {
		t.Fatalf("DeleteReviewByID: %v", err)
	}

	_, err = r.GetReviewByID(ctx, id)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("после удаления ожидалась ErrNotFound, получено: %v", err)
	}
}
