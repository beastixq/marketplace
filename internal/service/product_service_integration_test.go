//go:build integration

package service_test

import (
	"context"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/shopspring/decimal"
)

func TestProductService_UpdateProduct_Integration(t *testing.T) {
	t.Run("price history categories and atomic rollback", func(t *testing.T) {
		f := newIntegrationFixture(t)
		actor, sellerID := f.seller()
		id, categoryID := f.product(actor, sellerID, 5), f.category()
		ctx := context.Background()
		price, categories := decimal.NewFromInt(125), []int64{categoryID}
		changedBy := "integration-seller"

		updated, err := f.products.UpdateProduct(ctx, actor, id, m.ProductUpdate{Price: &price, CategoryIDs: &categories, ChangedBy: &changedBy})
		mustIntegration(t, err)
		if !updated.Price.Equal(price) {
			t.Fatalf("price = %s", updated.Price)
		}
		history, err := f.products.GetProductPriceHistory(ctx, id, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
		mustIntegration(t, err)
		if len(history) != 1 || !history[0].OldPrice.Equal(decimal.NewFromInt(100)) || !history[0].NewPrice.Equal(price) {
			t.Fatalf("history = %+v", history)
		}
		gotCategories, err := f.products.GetProductCategories(ctx, id)
		mustIntegration(t, err)
		if len(gotCategories) != 1 || gotCategories[0].ID != categoryID {
			t.Fatalf("categories = %+v", gotCategories)
		}
		invalidCategories, nextPrice := []int64{-1}, decimal.NewFromInt(200)
		if _, err := f.products.UpdateProduct(ctx, actor, id, m.ProductUpdate{Price: &nextPrice, CategoryIDs: &invalidCategories, ChangedBy: &changedBy}); err == nil {
			t.Fatal("invalid category update succeeded")
		}
		unchanged, err := f.products.GetProductByID(ctx, id)
		mustIntegration(t, err)
		if !unchanged.Price.Equal(price) {
			t.Fatal("price update escaped rollback")
		}
		history, err = f.products.GetProductPriceHistory(ctx, id, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
		mustIntegration(t, err)
		if len(history) != 1 {
			t.Fatal("history update escaped rollback")
		}
		mustIntegration(t, repo.NewProductRepo(f.pool).ChangeStockAndReserved(ctx, id, 0, 2))
		stock := 1
		if _, err := f.products.UpdateProduct(ctx, actor, id, m.ProductUpdate{StockQuantity: &stock}); err == nil {
			t.Fatal("stock below reserve accepted")
		}
		f.stock(id, 5, 2)
	})
}
