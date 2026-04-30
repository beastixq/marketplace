package repository

import (
	"context"
	"fmt"

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
	reviewRepo := store.NewReviewRepo(pool)
	return &Component{
		Pool:           pool,
		User:           store.NewUserRepo(pool),
		Seller:         store.NewSellerRepo(pool),
		Address:        store.NewAddressRepo(pool),
		Review:         reviewRepo,
		ReviewPurchase: reviewRepo,
		Product:        store.NewProductRepo(pool),
		Order:          store.NewOrderRepo(pool),
		OrderItem:      store.NewOrderItemRepo(pool),
		Category:       store.NewCategoryRepo(pool),
		Backoffice:     store.NewBackofficeRepo(pool),
		TxManager:      store.NewPgxTxManager(pool),
	}
}

func (c *Component) Close() {
	if c != nil && c.Pool != nil {
		c.Pool.Close()
	}
}
