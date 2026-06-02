package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/beastixq/marketplace/internal/cache"
	store "github.com/beastixq/marketplace/internal/repository"
	svc "github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Component struct {
	Pool           *pgxpool.Pool
	User           svc.UserRepo
	Seller         svc.SellerRepo
	Address        svc.AddressRepo
	Review         svc.ReviewRepo
	ReviewPurchase svc.ReviewPurchaseChecker
	Product        svc.ProductRepo
	Order          svc.OrderRepo
	OrderItem      svc.OrderItemRepo
	Category       svc.CategoryRepo
	Backoffice     svc.BackofficeRepo
	TxManager      svc.TxManager
}

func New(ctx context.Context, dbURL string) (*Component, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return NewFromPool(pool), nil
}

func NewFromPool(pool *pgxpool.Pool) *Component {
	return NewFromPoolWithCache(pool, nil)
}

// CacheConfig holds optional Redis caching parameters.
type CacheConfig struct {
	Client      *redis.Client
	ProductTTL  time.Duration
	CatalogTTL  time.Duration
	CategoryTTL time.Duration
	ReviewTTL   time.Duration
}

// NewFromPoolWithCache builds Component, optionally wrapping repos with cache
// decorators when cfg is non-nil and cfg.Client is set.
func NewFromPoolWithCache(pool *pgxpool.Pool, cfg *CacheConfig) *Component {
	baseReviewRepo := store.NewReviewRepo(pool)
	var reviewRepo svc.ReviewRepo = baseReviewRepo
	var categoryRepo svc.CategoryRepo = store.NewCategoryRepo(pool)

	var productRepo svc.ProductRepo = store.NewProductRepo(pool)
	if cfg != nil && cfg.Client != nil {
		productRepo = cache.NewProductRepoCache(productRepo, cfg.Client, cfg.ProductTTL, cfg.CatalogTTL)
		reviewRepo = cache.NewReviewRepoCache(reviewRepo, cfg.Client, cfg.ReviewTTL)
		categoryRepo = cache.NewCategoryRepoCache(categoryRepo, cfg.Client, cfg.CategoryTTL)
	}

	return &Component{
		Pool:           pool,
		User:           store.NewUserRepo(pool),
		Seller:         store.NewSellerRepo(pool),
		Address:        store.NewAddressRepo(pool),
		Review:         reviewRepo,
		ReviewPurchase: baseReviewRepo,
		Product:        productRepo,
		Order:          store.NewOrderRepo(pool),
		OrderItem:      store.NewOrderItemRepo(pool),
		Category:       categoryRepo,
		Backoffice:     store.NewBackofficeRepo(pool),
		TxManager:      store.NewPgxTxManager(pool),
	}
}

func (c *Component) Close() {
	if c != nil && c.Pool != nil {
		c.Pool.Close()
	}
}
