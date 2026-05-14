package repository_test

import (
	"context"
	"testing"
	"time"

	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
)

var _ service.FavoriteRepo = repo.FavoriteRepoImpl{}

// cleanupFavorite removes a (user_id, product_id) row after the test.
func cleanupFavorite(t *testing.T, userID, productID int64) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM favorites WHERE user_id = $1 AND product_id = $2", userID, productID)
	})
}

// ----- US1: Add -----

func TestFavoriteRepo_Add_Inserts(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	userID := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()
	cleanupFavorite(t, userID, productID)

	created, err := r.Add(ctx, userID, productID)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !created {
		t.Fatalf("expected created=true on first insert")
	}
}

func TestFavoriteRepo_Add_IdempotentOnConflict(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	userID := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()
	cleanupFavorite(t, userID, productID)

	if _, err := r.Add(ctx, userID, productID); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	created, err := r.Add(ctx, userID, productID)
	if err != nil {
		t.Fatalf("second Add: %v", err)
	}
	if created {
		t.Fatalf("expected created=false on second insert")
	}

	var count int
	if err := testPool.QueryRow(ctx, "SELECT count(*) FROM favorites WHERE user_id = $1 AND product_id = $2", userID, productID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 row, got %d", count)
	}
}

func TestFavoriteRepo_Add_ProductFKViolation(t *testing.T) {
	userID := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()

	_, err := r.Add(ctx, userID, 9_999_999_999)
	if err == nil {
		t.Fatalf("expected error for non-existent product, got nil")
	}
	// repo translates FK violation to service.ErrProductNotFound
	if err != service.ErrProductNotFound { //nolint:errorlint
		t.Fatalf("expected service.ErrProductNotFound, got %v", err)
	}
}

// ----- US2: List -----

func TestFavoriteRepo_List_OrderedNewestFirst(t *testing.T) {
	sellerID := createTestSeller(t)
	p1 := createTestProduct(t, sellerID)
	p2 := createTestProduct(t, sellerID)
	userID := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()
	cleanupFavorite(t, userID, p1)
	cleanupFavorite(t, userID, p2)

	if _, err := r.Add(ctx, userID, p1); err != nil {
		t.Fatalf("Add p1: %v", err)
	}
	// Ensure a strictly later created_at for p2.
	time.Sleep(10 * time.Millisecond)
	if _, err := r.Add(ctx, userID, p2); err != nil {
		t.Fatalf("Add p2: %v", err)
	}

	items, total, err := r.List(ctx, userID, 0, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("expected 2 items/total=2, got len=%d total=%d", len(items), total)
	}
	if items[0].Product.ID != p2 {
		t.Errorf("expected p2 first (newest), got %d", items[0].Product.ID)
	}
	if items[1].Product.ID != p1 {
		t.Errorf("expected p1 second, got %d", items[1].Product.ID)
	}
}

func TestFavoriteRepo_List_Pagination(t *testing.T) {
	sellerID := createTestSeller(t)
	p1 := createTestProduct(t, sellerID)
	p2 := createTestProduct(t, sellerID)
	p3 := createTestProduct(t, sellerID)
	userID := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()
	cleanupFavorite(t, userID, p1)
	cleanupFavorite(t, userID, p2)
	cleanupFavorite(t, userID, p3)

	for _, pid := range []int64{p1, p2, p3} {
		if _, err := r.Add(ctx, userID, pid); err != nil {
			t.Fatalf("Add: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	page1, total, err := r.List(ctx, userID, 0, 2)
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	if len(page1) != 2 {
		t.Errorf("expected page size 2, got %d", len(page1))
	}

	page2, _, err := r.List(ctx, userID, 2, 2)
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("expected 1 item on page2, got %d", len(page2))
	}
}

func TestFavoriteRepo_List_CrossUserIsolation(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	userA := createTestUser(t)
	userB := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()
	cleanupFavorite(t, userA, productID)

	if _, err := r.Add(ctx, userA, productID); err != nil {
		t.Fatalf("Add A: %v", err)
	}

	itemsB, totalB, err := r.List(ctx, userB, 0, 10)
	if err != nil {
		t.Fatalf("List B: %v", err)
	}
	if totalB != 0 || len(itemsB) != 0 {
		t.Fatalf("B must see 0 favorites, got len=%d total=%d", len(itemsB), totalB)
	}
}

func TestFavoriteRepo_List_EmptyForUnusedUser(t *testing.T) {
	userID := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()

	items, total, err := r.List(ctx, userID, 0, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("expected empty list, got len=%d total=%d", len(items), total)
	}
}

// ----- US3: Remove -----

func TestFavoriteRepo_Remove_ExistingRow(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	userID := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()

	if _, err := r.Add(ctx, userID, productID); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := r.Remove(ctx, userID, productID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	var count int
	if err := testPool.QueryRow(ctx, "SELECT count(*) FROM favorites WHERE user_id = $1 AND product_id = $2", userID, productID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected row removed, got count=%d", count)
	}
}

func TestFavoriteRepo_Remove_NonExistingRow_NoError(t *testing.T) {
	userID := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()

	if err := r.Remove(ctx, userID, 9_999_999_999); err != nil {
		t.Fatalf("Remove non-existent: %v", err)
	}
}

func TestFavoriteRepo_Remove_DoesNotAffectOtherUsers(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	userA := createTestUser(t)
	userB := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()
	cleanupFavorite(t, userB, productID)

	if _, err := r.Add(ctx, userA, productID); err != nil {
		t.Fatalf("Add A: %v", err)
	}
	if _, err := r.Add(ctx, userB, productID); err != nil {
		t.Fatalf("Add B: %v", err)
	}

	if err := r.Remove(ctx, userA, productID); err != nil {
		t.Fatalf("Remove A: %v", err)
	}

	favorited, _, err := r.Exists(ctx, userB, productID)
	if err != nil {
		t.Fatalf("Exists B: %v", err)
	}
	if !favorited {
		t.Fatalf("B's favorite must survive A's removal")
	}
}

// ----- US4: Exists -----

func TestFavoriteRepo_Exists_TrueAndFalse(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	userID := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()
	cleanupFavorite(t, userID, productID)

	favorited, addedAt, err := r.Exists(ctx, userID, productID)
	if err != nil {
		t.Fatalf("Exists before add: %v", err)
	}
	if favorited || addedAt != nil {
		t.Fatalf("expected (false, nil) before add, got (%v, %v)", favorited, addedAt)
	}

	if _, err := r.Add(ctx, userID, productID); err != nil {
		t.Fatalf("Add: %v", err)
	}

	favorited, addedAt, err = r.Exists(ctx, userID, productID)
	if err != nil {
		t.Fatalf("Exists after add: %v", err)
	}
	if !favorited || addedAt == nil {
		t.Fatalf("expected (true, *time) after add, got (%v, %v)", favorited, addedAt)
	}
	if time.Since(*addedAt) > time.Minute {
		t.Errorf("addedAt should be recent, got %v", *addedAt)
	}
}

// ----- US5: Cascades -----

func TestFavoriteRepo_Cascade_OnProductDelete(t *testing.T) {
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	userA := createTestUser(t)
	userB := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()

	if _, err := r.Add(ctx, userA, productID); err != nil {
		t.Fatalf("Add A: %v", err)
	}
	if _, err := r.Add(ctx, userB, productID); err != nil {
		t.Fatalf("Add B: %v", err)
	}

	// Hard-delete: bypass any soft-delete logic in the product service; the
	// cascade FK must remove all favorite rows referencing this product.
	if _, err := testPool.Exec(ctx, "DELETE FROM product_price_history WHERE product_id = $1", productID); err != nil {
		t.Fatalf("clear price history: %v", err)
	}
	if _, err := testPool.Exec(ctx, "DELETE FROM products WHERE id = $1", productID); err != nil {
		t.Fatalf("delete product: %v", err)
	}

	var count int
	if err := testPool.QueryRow(ctx, "SELECT count(*) FROM favorites WHERE product_id = $1", productID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 favorites after product cascade, got %d", count)
	}
}

func TestFavoriteRepo_Cascade_OnUserDelete(t *testing.T) {
	sellerID := createTestSeller(t)
	p1 := createTestProduct(t, sellerID)
	p2 := createTestProduct(t, sellerID)
	userA := createTestUser(t)
	userB := createTestUser(t)
	r := repo.NewFavoriteRepo(testPool)
	ctx := context.Background()
	cleanupFavorite(t, userB, p1)

	if _, err := r.Add(ctx, userA, p1); err != nil {
		t.Fatalf("Add A.p1: %v", err)
	}
	if _, err := r.Add(ctx, userA, p2); err != nil {
		t.Fatalf("Add A.p2: %v", err)
	}
	if _, err := r.Add(ctx, userB, p1); err != nil {
		t.Fatalf("Add B.p1: %v", err)
	}

	if _, err := testPool.Exec(ctx, "DELETE FROM users WHERE id = $1", userA); err != nil {
		t.Fatalf("delete user A: %v", err)
	}

	var aCount, bCount int
	if err := testPool.QueryRow(ctx, "SELECT count(*) FROM favorites WHERE user_id = $1", userA).Scan(&aCount); err != nil {
		t.Fatalf("count A: %v", err)
	}
	if err := testPool.QueryRow(ctx, "SELECT count(*) FROM favorites WHERE user_id = $1", userB).Scan(&bCount); err != nil {
		t.Fatalf("count B: %v", err)
	}
	if aCount != 0 {
		t.Fatalf("expected A's favorites cleared by cascade, got %d", aCount)
	}
	if bCount != 1 {
		t.Fatalf("expected B's favorite untouched, got %d", bCount)
	}
}

// ----- US1 Role grants (Constitution III) -----

func TestFavoriteRepo_RoleGrants(t *testing.T) {
	roles := []string{"marketplace_buyer", "marketplace_seller", "marketplace_admin"}
	ctx := context.Background()

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			sellerID := createTestSeller(t)
			productID := createTestProduct(t, sellerID)
			userID := createTestUser(t)
			cleanupFavorite(t, userID, productID)

			// Open a transaction, switch to the role, run INSERT/SELECT/DELETE,
			// rollback so the role change does not leak.
			tx, err := testPool.Begin(ctx)
			if err != nil {
				t.Fatalf("begin: %v", err)
			}
			defer tx.Rollback(ctx) //nolint:errcheck

			if _, err := tx.Exec(ctx, "SET LOCAL ROLE "+role); err != nil {
				t.Fatalf("SET LOCAL ROLE %s: %v", role, err)
			}

			if _, err := tx.Exec(ctx, "INSERT INTO favorites (user_id, product_id) VALUES ($1, $2)", userID, productID); err != nil {
				t.Fatalf("INSERT as %s: %v", role, err)
			}
			var count int
			if err := tx.QueryRow(ctx, "SELECT count(*) FROM favorites WHERE user_id = $1 AND product_id = $2", userID, productID).Scan(&count); err != nil {
				t.Fatalf("SELECT as %s: %v", role, err)
			}
			if count != 1 {
				t.Fatalf("expected 1 row, got %d", count)
			}
			if _, err := tx.Exec(ctx, "DELETE FROM favorites WHERE user_id = $1 AND product_id = $2", userID, productID); err != nil {
				t.Fatalf("DELETE as %s: %v", role, err)
			}
		})
	}
}
