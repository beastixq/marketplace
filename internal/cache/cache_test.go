package cache

import (
	"context"
	"errors"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	svc "github.com/beastixq/marketplace/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

func redisClient(t *testing.T) *redis.Client {
	t.Helper()
	if _, err := exec.LookPath("redis-server"); err != nil {
		t.Skip("redis-server not installed")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve redis port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cmd := exec.Command(
		"redis-server",
		"--bind", "127.0.0.1",
		"--port", strconv.Itoa(port),
		"--save", "",
		"--appendonly", "no",
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start redis-server: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:" + strconv.Itoa(port)})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := client.Ping(context.Background()).Err(); err == nil {
			t.Cleanup(func() { _ = client.Close() })
			return client
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("redis-server did not become ready")
	return nil
}

func TestReadableCacheKeysAreCanonical(t *testing.T) {
	name := "phone case"
	minPrice := decimal.NewFromInt(10)
	maxPrice := decimal.NewFromInt(99)
	minPriceWithScale, err := decimal.NewFromString("10.00")
	if err != nil {
		t.Fatalf("decimal fixture: %v", err)
	}
	sortOrder := m.SortingOrderAsc
	page := m.PaginationOpts{Page: 1, Limit: 12}

	key1 := ProductCatalogKey(m.CatalogOptions{
		Categories:   []string{"Accessories", "Phones"},
		FilterName:   &name,
		MinPrice:     &minPrice,
		MaxPrice:     &maxPrice,
		Pagination:   &page,
		SortingOrder: &sortOrder,
	})
	key2 := ProductCatalogKey(m.CatalogOptions{
		Categories:   []string{"Phones", "Accessories"},
		FilterName:   &name,
		MinPrice:     &minPrice,
		MaxPrice:     &maxPrice,
		Pagination:   &page,
		SortingOrder: &sortOrder,
	})

	if key1 != key2 {
		t.Fatalf("catalog keys differ for equivalent category sets:\n%s\n%s", key1, key2)
	}
	key3 := ProductCatalogKey(m.CatalogOptions{
		MinPrice:   &minPriceWithScale,
		Pagination: &page,
	})
	key4 := ProductCatalogKey(m.CatalogOptions{
		MinPrice:   &minPrice,
		Pagination: &page,
	})
	if key3 != key4 {
		t.Fatalf("catalog keys differ for equivalent decimal values:\n%s\n%s", key3, key4)
	}
	if !strings.HasPrefix(key1, ProductCatalogPrefix()) {
		t.Fatalf("catalog key %q does not use readable prefix", key1)
	}
	for _, part := range []string{"category=Accessories", "category=Phones", "filter_name=phone+case", "limit=12", "page=1", "sort=asc"} {
		if !strings.Contains(key1, part) {
			t.Fatalf("catalog key %q missing %q", key1, part)
		}
	}
}

func TestProductRepoCacheCatalogHitAndInvalidation(t *testing.T) {
	ctx := context.Background()
	rdb := redisClient(t)
	repo := &fakeProductRepo{
		product:  product(1, "Phone"),
		products: []m.Product{product(1, "Phone")},
	}
	cache := NewProductRepoCache(repo, rdb, time.Hour, time.Hour)
	opts := m.CatalogOptions{Pagination: &m.PaginationOpts{Page: 1, Limit: 12}}

	if _, err := cache.GetProducts(ctx, opts); err != nil {
		t.Fatalf("first GetProducts: %v", err)
	}
	if _, err := cache.GetProducts(ctx, opts); err != nil {
		t.Fatalf("second GetProducts: %v", err)
	}
	if repo.productsCalls != 1 {
		t.Fatalf("GetProducts loader calls: got %d, want 1", repo.productsCalls)
	}

	if _, err := cache.GetProductByID(ctx, 1); err != nil {
		t.Fatalf("GetProductByID: %v", err)
	}
	if _, err := cache.UpdateProduct(ctx, 1, m.ProductUpdate{Name: strPtr("Updated")}); err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}
	assertRedisMissing(t, rdb, ProductByIDKey(1))
	assertRedisMissing(t, rdb, ProductCatalogKey(opts))
}

func TestProductRepoCacheStockChangeInvalidatesProductOnly(t *testing.T) {
	ctx := context.Background()
	rdb := redisClient(t)
	repo := &fakeProductRepo{product: product(1, "Phone")}
	cache := NewProductRepoCache(repo, rdb, time.Hour, time.Hour)
	catalogOpts := m.CatalogOptions{Pagination: &m.PaginationOpts{Page: 1, Limit: 12}}

	if err := rdb.Set(ctx, ProductByIDKey(1), "{}", time.Hour).Err(); err != nil {
		t.Fatalf("seed product key: %v", err)
	}
	if err := rdb.Set(ctx, ProductCatalogKey(catalogOpts), "[]", time.Hour).Err(); err != nil {
		t.Fatalf("seed catalog key: %v", err)
	}

	if err := cache.ChangeStockAndReserved(ctx, 1, 0, 1); err != nil {
		t.Fatalf("ChangeStockAndReserved: %v", err)
	}

	assertRedisMissing(t, rdb, ProductByIDKey(1))
	assertRedisExists(t, rdb, ProductCatalogKey(catalogOpts))
}

func TestCategoryRepoCacheInvalidatesCategoriesAndCatalog(t *testing.T) {
	ctx := context.Background()
	rdb := redisClient(t)
	repo := &fakeCategoryRepo{categories: []m.Category{{ID: 1, Name: "Phones"}}}
	cache := NewCategoryRepoCache(repo, rdb, time.Hour)
	categoryOpts := m.CategoryListOptions{Pagination: m.PaginationOpts{Page: 1, Limit: 50}}
	catalogOpts := m.CatalogOptions{Pagination: &m.PaginationOpts{Page: 1, Limit: 12}}

	if _, err := cache.GetCategories(ctx, categoryOpts); err != nil {
		t.Fatalf("GetCategories: %v", err)
	}
	if err := rdb.Set(ctx, ProductCatalogKey(catalogOpts), "[]", time.Hour).Err(); err != nil {
		t.Fatalf("seed product catalog key: %v", err)
	}
	if _, err := cache.CreateCategory(ctx, m.CategoryCreate{Name: "Books"}); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	assertRedisMissing(t, rdb, CategoryListKey(categoryOpts))
	assertRedisMissing(t, rdb, ProductCatalogKey(catalogOpts))
}

func TestReviewRepoCacheInvalidatesProductReadModels(t *testing.T) {
	ctx := context.Background()
	rdb := redisClient(t)
	repo := &fakeReviewRepo{reviews: []m.Review{{ID: 1, ProductID: 1, Rating: 5}}}
	cache := NewReviewRepoCache(repo, rdb, time.Hour)
	reviewOpts := m.PaginationOpts{Page: 1, Limit: 50}
	catalogOpts := m.CatalogOptions{Pagination: &m.PaginationOpts{Page: 1, Limit: 12}}

	if _, err := cache.GetReviewsByProductID(ctx, 1, reviewOpts); err != nil {
		t.Fatalf("GetReviewsByProductID: %v", err)
	}
	if err := rdb.Set(ctx, ProductByIDKey(1), "{}", time.Hour).Err(); err != nil {
		t.Fatalf("seed product key: %v", err)
	}
	if err := rdb.Set(ctx, ProductCatalogKey(catalogOpts), "[]", time.Hour).Err(); err != nil {
		t.Fatalf("seed catalog key: %v", err)
	}
	if _, err := cache.CreateReview(ctx, m.ReviewCreate{ProductID: 1, Rating: 5}); err != nil {
		t.Fatalf("CreateReview: %v", err)
	}

	assertRedisMissing(t, rdb, ProductReviewsKey(1, reviewOpts))
	assertRedisMissing(t, rdb, ProductByIDKey(1))
	assertRedisMissing(t, rdb, ProductCatalogKey(catalogOpts))
}

func TestReviewPrefixInvalidationDoesNotTouchOtherProducts(t *testing.T) {
	ctx := context.Background()
	rdb := redisClient(t)
	productFiveReviews := ProductReviewsKey(5, m.PaginationOpts{Page: 1, Limit: 10})
	productFiftyReviews := ProductReviewsKey(50, m.PaginationOpts{Page: 1, Limit: 10})

	if err := rdb.Set(ctx, productFiveReviews, "[]", time.Hour).Err(); err != nil {
		t.Fatalf("seed product 5 reviews key: %v", err)
	}
	if err := rdb.Set(ctx, productFiftyReviews, "[]", time.Hour).Err(); err != nil {
		t.Fatalf("seed product 50 reviews key: %v", err)
	}

	deleteByPrefix(ctx, rdb, ProductReviewsPrefix(5))

	assertRedisMissing(t, rdb, productFiveReviews)
	assertRedisExists(t, rdb, productFiftyReviews)
}

func TestInvalidationDefersUntilAfterCommitHook(t *testing.T) {
	ctx := context.Background()
	rdb := redisClient(t)
	txCtx, hooks := svc.WithAfterCommitHooks(ctx)
	if err := rdb.Set(ctx, ProductByIDKey(7), "{}", time.Hour).Err(); err != nil {
		t.Fatalf("seed product key: %v", err)
	}

	invalidateAfterCommit(txCtx, func(ctx context.Context) {
		deleteKeys(ctx, rdb, ProductByIDKey(7))
	})
	assertRedisExists(t, rdb, ProductByIDKey(7))

	hooks.Run(ctx)
	assertRedisMissing(t, rdb, ProductByIDKey(7))
}

func TestTokenBlocklistExpires(t *testing.T) {
	ctx := context.Background()
	rdb := redisClient(t)
	blocklist := NewTokenBlocklist(rdb)

	if err := blocklist.Add(ctx, "token-id", 100*time.Millisecond); err != nil {
		t.Fatalf("Add: %v", err)
	}
	blocked, err := blocklist.Contains(ctx, "token-id")
	if err != nil {
		t.Fatalf("Contains: %v", err)
	}
	if !blocked {
		t.Fatal("Contains before TTL: got false, want true")
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		blocked, err = blocklist.Contains(ctx, "token-id")
		if err != nil {
			t.Fatalf("Contains after TTL: %v", err)
		}
		if !blocked {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("token blocklist key did not expire")
}

func TestGetOrLoadDoesNotWriteNegativeCache(t *testing.T) {
	ctx := context.Background()
	rdb := redisClient(t)
	key := "negative-cache-test"
	loaderErr := errors.New("loader failed")

	_, err := GetOrLoad(ctx, rdb, key, time.Hour, func(context.Context) (string, error) {
		return "", loaderErr
	})
	if !errors.Is(err, loaderErr) {
		t.Fatalf("GetOrLoad error: got %v, want %v", err, loaderErr)
	}
	assertRedisMissing(t, rdb, key)
}

func TestGetOrLoadFallsBackOnRedisError(t *testing.T) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 20 * time.Millisecond})
	defer rdb.Close()

	calls := 0
	got, err := GetOrLoad(ctx, rdb, "broken", time.Hour, func(context.Context) (string, error) {
		calls++
		return "from loader", nil
	})
	if err != nil {
		t.Fatalf("GetOrLoad: %v", err)
	}
	if got != "from loader" || calls != 1 {
		t.Fatalf("fallback got (%q, calls=%d), want loader once", got, calls)
	}
}

func assertRedisExists(t *testing.T, rdb *redis.Client, key string) {
	t.Helper()
	exists, err := rdb.Exists(context.Background(), key).Result()
	if err != nil {
		t.Fatalf("Exists %q: %v", key, err)
	}
	if exists == 0 {
		t.Fatalf("Redis key %q: got missing, want exists", key)
	}
}

func assertRedisMissing(t *testing.T, rdb *redis.Client, key string) {
	t.Helper()
	exists, err := rdb.Exists(context.Background(), key).Result()
	if err != nil {
		t.Fatalf("Exists %q: %v", key, err)
	}
	if exists != 0 {
		t.Fatalf("Redis key %q: got exists, want missing", key)
	}
}

func product(id int64, name string) m.Product {
	return m.Product{ID: id, SellerID: 1, Name: name, Price: decimal.NewFromInt(100), StockQuantity: 10}
}

func strPtr(v string) *string { return &v }

type fakeProductRepo struct {
	product       m.Product
	products      []m.Product
	productsCalls int
}

func (f *fakeProductRepo) GetProducts(context.Context, m.CatalogOptions) ([]m.Product, error) {
	f.productsCalls++
	return f.products, nil
}

func (f *fakeProductRepo) GetProductByID(context.Context, int64) (m.Product, error) {
	return f.product, nil
}

func (f *fakeProductRepo) GetProductByIDForUpdate(context.Context, int64) (m.Product, error) {
	return f.product, nil
}

func (f *fakeProductRepo) GetProductPriceHistory(context.Context, int64, time.Time, time.Time) ([]m.ProductPriceHistory, error) {
	return nil, nil
}

func (f *fakeProductRepo) CreateProduct(context.Context, m.ProductCreate) (int64, error) {
	return f.product.ID, nil
}

func (f *fakeProductRepo) UpdateProduct(_ context.Context, id int64, pu m.ProductUpdate) (m.Product, error) {
	if pu.Name != nil {
		f.product.Name = *pu.Name
	}
	f.product.ID = id
	return f.product, nil
}

func (f *fakeProductRepo) ChangeStockAndReserved(context.Context, int64, int, int) error {
	return nil
}

func (f *fakeProductRepo) DeleteProductByID(context.Context, int64) error {
	return nil
}

func (f *fakeProductRepo) GetProductCategories(context.Context, int64) ([]m.Category, error) {
	return nil, nil
}

func (f *fakeProductRepo) ReplaceProductCategories(context.Context, int64, []int64) error {
	return nil
}

type fakeCategoryRepo struct {
	categories []m.Category
}

func (f *fakeCategoryRepo) GetCategories(context.Context, m.CategoryListOptions) ([]m.Category, error) {
	return f.categories, nil
}

func (f *fakeCategoryRepo) GetCategoryByID(context.Context, int64) (m.Category, error) {
	if len(f.categories) == 0 {
		return m.Category{}, errors.New("not found")
	}
	return f.categories[0], nil
}

func (f *fakeCategoryRepo) CreateCategory(context.Context, m.CategoryCreate) (int64, error) {
	return 1, nil
}

func (f *fakeCategoryRepo) UpdateCategory(context.Context, int64, m.CategoryUpdate) (m.Category, error) {
	return m.Category{ID: 1, Name: "Updated"}, nil
}

func (f *fakeCategoryRepo) DeleteCategoryByID(context.Context, int64) error {
	return nil
}

type fakeReviewRepo struct {
	reviews []m.Review
}

func (f *fakeReviewRepo) GetReviewByID(context.Context, int64) (m.Review, error) {
	if len(f.reviews) == 0 {
		return m.Review{}, svc.ErrNotFound
	}
	return f.reviews[0], nil
}

func (f *fakeReviewRepo) GetReviewsByProductID(context.Context, int64, m.PaginationOpts) ([]m.Review, error) {
	return f.reviews, nil
}

func (f *fakeReviewRepo) CreateReview(context.Context, m.ReviewCreate) (int64, error) {
	return 1, nil
}

func (f *fakeReviewRepo) UpdateReview(context.Context, int64, m.ReviewUpdate) (m.Review, error) {
	if len(f.reviews) == 0 {
		return m.Review{}, svc.ErrNotFound
	}
	return f.reviews[0], nil
}

func (f *fakeReviewRepo) DeleteReviewByID(context.Context, int64) error {
	return nil
}

func (f *fakeReviewRepo) UserPurchasedProduct(context.Context, int64, int64) (bool, error) {
	return true, nil
}
