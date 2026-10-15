//go:build integration

package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	s "github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

func (f *serviceIntegration) createProduct(actor s.Actor, sellerID int64, stock int) int64 {
	f.t.Helper()
	id, err := f.productService().CreateProduct(context.Background(), actor, m.ProductCreate{
		SellerID: sellerID, Name: "Integration product", Price: decimal.NewFromInt(100), StockQuantity: stock,
	})
	mustIntegration(f.t, err)
	return id
}

func assertPersistedStock(t *testing.T, pool *pgxpool.Pool, productID int64, stock, reserved int) {
	t.Helper()
	product, err := repo.NewProductRepo(pool).GetProductByID(context.Background(), productID)
	mustIntegration(t, err)
	if product.StockQuantity != stock || product.ReservedQuantity != reserved {
		t.Fatalf("product %d stock/reserved = %d/%d, want %d/%d", productID, product.StockQuantity, product.ReservedQuantity, stock, reserved)
	}
}

func TestProductService_UpdateProduct_Integration(t *testing.T) {
	t.Run("price history categories and atomic rollback", func(t *testing.T) {
		f := newServiceIntegration(t)
		actor, sellerID := f.createSeller()
		id, categoryID := f.createProduct(actor, sellerID, 5), f.createCategory()
		ctx := context.Background()
		price, categories := decimal.NewFromInt(125), []int64{categoryID}
		changedBy := "integration-seller"

		updated, err := f.productService().UpdateProduct(ctx, actor, id, m.ProductUpdate{Price: &price, CategoryIDs: &categories, ChangedBy: &changedBy})
		mustIntegration(t, err)
		if !updated.Price.Equal(price) {
			t.Fatalf("price = %s", updated.Price)
		}
		history, err := f.productService().GetProductPriceHistory(ctx, id, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
		mustIntegration(t, err)
		if len(history) != 1 || !history[0].OldPrice.Equal(decimal.NewFromInt(100)) || !history[0].NewPrice.Equal(price) {
			t.Fatalf("history = %+v", history)
		}
		gotCategories, err := f.productService().GetProductCategories(ctx, id)
		mustIntegration(t, err)
		if len(gotCategories) != 1 || gotCategories[0].ID != categoryID {
			t.Fatalf("categories = %+v", gotCategories)
		}
		invalidCategories, nextPrice := []int64{-1}, decimal.NewFromInt(200)
		if _, err := f.productService().UpdateProduct(ctx, actor, id, m.ProductUpdate{Price: &nextPrice, CategoryIDs: &invalidCategories, ChangedBy: &changedBy}); err == nil {
			t.Fatal("invalid category update succeeded")
		}
		unchanged, err := f.productService().GetProductByID(ctx, id)
		mustIntegration(t, err)
		if !unchanged.Price.Equal(price) {
			t.Fatal("price update escaped rollback")
		}
		history, err = f.productService().GetProductPriceHistory(ctx, id, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
		mustIntegration(t, err)
		if len(history) != 1 {
			t.Fatal("history update escaped rollback")
		}
		gotCategories, err = f.productService().GetProductCategories(ctx, id)
		mustIntegration(t, err)
		if len(gotCategories) != 1 || gotCategories[0].ID != categoryID {
			t.Fatalf("categories escaped rollback: %+v", gotCategories)
		}
		buyer := f.createActor(m.RoleBuyer)
		createPendingOrder(t, f.orderService(), buyer, id, f.createAddress(buyer))
		stock := 0
		if _, err := f.productService().UpdateProduct(ctx, actor, id, m.ProductUpdate{StockQuantity: &stock}); !errors.Is(err, s.ErrStockBelowReserved) {
			t.Fatalf("stock below reserve = %v", err)
		}
		assertPersistedStock(t, f.pool, id, 5, 1)
	})
}
