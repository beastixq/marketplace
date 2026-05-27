package mongorepo

import (
	"context"
	"os"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
)

func TestMongoServiceSmoke(t *testing.T) {
	uri, ok := os.LookupEnv("MONGODB_URI")
	if !ok || uri == "" {
		t.Skip("MONGODB_URI not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := New(ctx, uri, "marketplace_mongo_service_test_"+time.Now().Format("20060102150405"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer store.Close(context.Background())
	defer store.db.Drop(context.Background())

	userSvc := service.NewUserService(store, 4)
	sellerSvc := service.NewSellerService(store)
	addressSvc := service.NewAddressService(store)
	productSvc := service.NewProductService(store, store, store, store)
	orderSvc := service.NewOrderService(store, store, store, store, store, store)
	reviewSvc := service.NewReviewService(store, store, store)
	favoriteSvc := service.NewFavoriteService(store, store)
	backofficeSvc := service.NewBackofficeService(store)

	buyerID, err := userSvc.CreateUser(ctx, m.UserCreate{
		Email:    "buyer-smoke@example.test",
		Password: "secret",
		FullName: "Buyer Smoke",
		Role:     m.RoleBuyer,
	})
	if err != nil {
		t.Fatalf("Create buyer: %v", err)
	}
	sellerUserID, err := userSvc.CreateUser(ctx, m.UserCreate{
		Email:    "seller-smoke@example.test",
		Password: "secret",
		FullName: "Seller Smoke",
		Role:     m.RoleSeller,
	})
	if err != nil {
		t.Fatalf("Create seller user: %v", err)
	}

	buyer := service.Actor{UserID: buyerID, Role: m.RoleBuyer}
	sellerActor := service.Actor{UserID: sellerUserID, Role: m.RoleSeller}
	admin := service.Actor{UserID: 999, Role: m.RoleAdmin}

	sellerID, err := sellerSvc.CreateSeller(ctx, sellerActor, m.SellerCreate{CompanyName: "Smoke Seller"})
	if err != nil {
		t.Fatalf("CreateSeller: %v", err)
	}
	productID, err := productSvc.CreateProduct(ctx, sellerActor, m.ProductCreate{
		SellerID:      sellerID,
		Name:          "Smoke Product",
		Price:         decimal.NewFromInt(50),
		StockQuantity: 3,
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	addressID, err := addressSvc.CreateAddress(ctx, buyer, m.AddressCreate{
		City:    "Moscow",
		Street:  "Baumanskaya",
		House:   "5",
		ZipCode: "105005",
	})
	if err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}

	if err := orderSvc.AddItemToCart(ctx, buyer, productID, 2); err != nil {
		t.Fatalf("AddItemToCart: %v", err)
	}
	orderIDs, err := orderSvc.Checkout(ctx, buyer, addressID)
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if len(orderIDs) != 1 {
		t.Fatalf("Checkout order count: got %d", len(orderIDs))
	}
	if err := orderSvc.PayOrder(ctx, buyer, orderIDs[0]); err != nil {
		t.Fatalf("PayOrder: %v", err)
	}
	if err := orderSvc.ShipOrder(ctx, sellerActor, orderIDs[0]); err != nil {
		t.Fatalf("ShipOrder: %v", err)
	}
	if _, err := reviewSvc.CreateReview(ctx, buyer, m.ReviewCreate{ProductID: productID, Rating: 5}); err != nil {
		t.Fatalf("CreateReview: %v", err)
	}
	if created, err := favoriteSvc.AddFavorite(ctx, buyer, productID); err != nil || !created {
		t.Fatalf("AddFavorite: created=%v err=%v", created, err)
	}
	stats, err := backofficeSvc.GetPlatformStats(ctx, admin)
	if err != nil {
		t.Fatalf("GetPlatformStats: %v", err)
	}
	if stats.TotalRevenue.IsZero() {
		t.Fatal("GetPlatformStats: expected non-zero revenue")
	}
}
