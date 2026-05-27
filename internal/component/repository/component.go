package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/beastixq/marketplace/internal/cache"
	"github.com/beastixq/marketplace/internal/config"
	store "github.com/beastixq/marketplace/internal/repository"
	mongostore "github.com/beastixq/marketplace/internal/repository/mongo"
	svc "github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Component struct {
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
	Favorite       svc.FavoriteRepo
	TxManager      svc.TxManager

	closeFn func()
}

func New(ctx context.Context, dbURL string) (*Component, error) {
	return NewPostgres(ctx, dbURL, nil)
}

func NewFromConfig(ctx context.Context, cfg config.DatabaseConfig, cacheCfg *CacheConfig) (*Component, error) {
	switch cfg.Type {
	case "", config.DatabasePostgres:
		return NewPostgres(ctx, cfg.DSN, cacheCfg)
	case config.DatabaseMongo:
		return NewMongo(ctx, cfg.Mongo, cacheCfg)
	default:
		return nil, fmt.Errorf("unsupported database type %q", cfg.Type)
	}
}

func NewPostgres(ctx context.Context, dbURL string, cacheCfg *CacheConfig) (*Component, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return NewFromPoolWithCache(pool, cacheCfg), nil
}

func NewMongo(ctx context.Context, cfg config.MongoConfig, cacheCfg *CacheConfig) (*Component, error) {
	mongoStore, err := mongostore.New(ctx, cfg.URI, cfg.Name)
	if err != nil {
		return nil, err
	}

	var productRepo svc.ProductRepo = mongoStore
	if cacheCfg != nil && cacheCfg.Client != nil {
		productRepo = cache.NewProductRepoCache(productRepo, cacheCfg.Client, cacheCfg.ProductTTL)
	}

	return &Component{
		User:           mongoStore,
		Seller:         mongoStore,
		Address:        mongoStore,
		Review:         mongoStore,
		ReviewPurchase: mongoStore,
		Product:        productRepo,
		Order:          mongoStore,
		OrderItem:      mongoStore,
		Category:       mongoStore,
		Backoffice:     mongoStore,
		Favorite:       mongoStore,
		TxManager:      mongoStore,
		closeFn: func() {
			_ = mongoStore.Close(context.Background())
		},
	}, nil
}

func NewFromPool(pool *pgxpool.Pool) *Component {
	return NewFromPoolWithCache(pool, nil)
}

// CacheConfig holds optional Redis caching parameters.
type CacheConfig struct {
	Client     *redis.Client
	ProductTTL time.Duration
}

// NewFromPoolWithCache builds Component, optionally wrapping repos with cache
// decorators when cfg is non-nil and cfg.Client is set.
func NewFromPoolWithCache(pool *pgxpool.Pool, cfg *CacheConfig) *Component {
	reviewRepo := store.NewReviewRepo(pool)

	var productRepo svc.ProductRepo = store.NewProductRepo(pool)
	if cfg != nil && cfg.Client != nil {
		productRepo = cache.NewProductRepoCache(productRepo, cfg.Client, cfg.ProductTTL)
	}

	return &Component{
		User:           store.NewUserRepo(pool),
		Seller:         store.NewSellerRepo(pool),
		Address:        store.NewAddressRepo(pool),
		Review:         reviewRepo,
		ReviewPurchase: reviewRepo,
		Product:        productRepo,
		Order:          store.NewOrderRepo(pool),
		OrderItem:      store.NewOrderItemRepo(pool),
		Category:       store.NewCategoryRepo(pool),
		Backoffice:     store.NewBackofficeRepo(pool),
		Favorite:       store.NewFavoriteRepo(pool),
		TxManager:      store.NewPgxTxManager(pool),
		closeFn:        pool.Close,
	}
}

func (c *Component) Close() {
	if c != nil && c.closeFn != nil {
		c.closeFn()
	}
}
