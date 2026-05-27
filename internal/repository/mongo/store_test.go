package mongorepo

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
)

func TestStoreIndexesAndTransactions(t *testing.T) {
	uri, ok := os.LookupEnv("MONGODB_URI")
	if !ok || uri == "" {
		t.Skip("MONGODB_URI not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := New(ctx, uri, "marketplace_mongo_test_"+time.Now().Format("20060102150405"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer store.Close(context.Background())
	defer store.db.Drop(context.Background())

	if _, err := store.GetUserByEmail(ctx, "admin@marketplace.local"); err != nil {
		t.Fatalf("system admin account: %v", err)
	}
	if _, err := store.GetUserByEmail(ctx, "analyst@marketplace.local"); err != nil {
		t.Fatalf("system analyst account: %v", err)
	}

	buyerID, err := store.CreateUser(ctx, m.UserCreate{
		Email:    "buyer@example.test",
		Password: "hash",
		FullName: "Buyer",
		Role:     m.RoleBuyer,
	})
	if err != nil {
		t.Fatalf("CreateUser buyer: %v", err)
	}
	sellerUserID, err := store.CreateUser(ctx, m.UserCreate{
		Email:    "seller@example.test",
		Password: "hash",
		FullName: "Seller User",
		Role:     m.RoleSeller,
	})
	if err != nil {
		t.Fatalf("CreateUser seller: %v", err)
	}
	sellerID, err := store.CreateSeller(ctx, m.SellerCreate{UserID: sellerUserID, CompanyName: "Seller"})
	if err != nil {
		t.Fatalf("CreateSeller: %v", err)
	}
	productID, err := store.CreateProduct(ctx, m.ProductCreate{
		SellerID:      sellerID,
		Name:          "Keyboard",
		Price:         decimal.NewFromInt(100),
		StockQuantity: 5,
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if err := store.ChangeStockAndReserved(ctx, productID, 0, 6); !errors.Is(err, service.ErrStockInvariantViolated) {
		t.Fatalf("ChangeStockAndReserved expected invariant error, got %v", err)
	}

	if _, err = store.CreateOrder(ctx, m.OrderCreate{UserID: buyerID, Status: m.StatusDraft, TotalAmount: decimal.Zero}); err != nil {
		t.Fatalf("CreateOrder draft: %v", err)
	}
	if _, err = store.CreateOrder(ctx, m.OrderCreate{UserID: buyerID, Status: m.StatusDraft, TotalAmount: decimal.Zero}); err == nil {
		t.Fatal("CreateOrder second draft: expected partial unique index error")
	}

	rollbackEmail := "rollback@example.test"
	err = store.WithTransaction(ctx, func(ctx context.Context) error {
		if _, err := store.CreateUser(ctx, m.UserCreate{
			Email:    rollbackEmail,
			Password: "hash",
			FullName: "Rollback",
			Role:     m.RoleBuyer,
		}); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("WithTransaction: expected rollback error")
	}
	if _, err = store.GetUserByEmail(ctx, rollbackEmail); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("GetUserByEmail after rollback: got %v", err)
	}
}
