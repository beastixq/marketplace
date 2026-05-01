package web

import (
	"bytes"
	"context"
	"testing"
	"time"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/beastixq/marketplace/internal/testsupport"
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"
)

func TestTemplatesParse(t *testing.T) {
	handler := NewWebHandler(
		service.ProductService{},
		service.CategoryService{},
		service.AuthService{},
		service.UserService{},
		service.OrderService{},
		service.AddressService{},
		service.SellerService{},
		service.ReviewService{},
		service.BackofficeService{},
		nil,
	)
	if len(handler.templates) == 0 {
		t.Fatal("templates: got 0, want parsed templates")
	}
}

func TestProductTemplateRendersCurrentUserReview(t *testing.T) {
	handler := NewWebHandler(
		service.ProductService{},
		service.CategoryService{},
		service.AuthService{},
		service.UserService{},
		service.OrderService{},
		service.AddressService{},
		service.SellerService{},
		service.ReviewService{},
		service.BackofficeService{},
		nil,
	)

	comment := "good"
	review := model.Review{
		ID:        1,
		UserID:    7,
		ProductID: 10,
		Rating:    4,
		Comment:   &comment,
		CreatedAt: time.Now(),
	}
	var out bytes.Buffer
	if err := handler.templates["product"].ExecuteTemplate(&out, "layout", map[string]any{
		"User":              &userInfo{UserID: 7, Role: "buyer", FullName: "Buyer"},
		"Product":           model.Product{ID: 10, SellerID: 20, Name: "Product", Price: decimal.NewFromInt(100), StockQuantity: 1, CreatedAt: time.Now()},
		"Seller":            model.Seller{ID: 20, CompanyName: "Seller"},
		"PriceRange":        "3m",
		"ReviewUserNames":   map[int64]string{7: "Buyer"},
		"Reviews":           []model.Review{review},
		"CurrentUserReview": &review,
	}); err != nil {
		t.Fatalf("render product template: %v", err)
	}
}

func TestProductTemplateRendersAdminCategoryPicker(t *testing.T) {
	handler := NewWebHandler(
		service.ProductService{},
		service.CategoryService{},
		service.AuthService{},
		service.UserService{},
		service.OrderService{},
		service.AddressService{},
		service.SellerService{},
		service.ReviewService{},
		service.BackofficeService{},
		nil,
	)

	selected := model.Category{ID: 1, Name: "Books"}
	option := model.Category{ID: 2, Name: "Electronics"}
	var out bytes.Buffer
	if err := handler.templates["product"].ExecuteTemplate(&out, "layout", map[string]any{
		"User":              &userInfo{UserID: 1, Role: "admin", FullName: "Admin"},
		"Product":           model.Product{ID: 10, SellerID: 20, Name: "Product", Price: decimal.NewFromInt(100), StockQuantity: 1, CreatedAt: time.Now()},
		"Seller":            model.Seller{ID: 20, CompanyName: "Seller"},
		"PriceRange":        "3m",
		"ProductCategories": []model.Category{selected},
		"CategoryPicker": categoryPickerData{
			SearchAction: "/products/10",
			Page:         1,
			Selected:     []model.Category{selected},
			Options:      []model.Category{option},
		},
	}); err != nil {
		t.Fatalf("render product template: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("Change Categories")) {
		t.Fatal("admin category picker was not rendered")
	}
}

func TestProfileTemplateRendersPasswordForm(t *testing.T) {
	handler := NewWebHandler(
		service.ProductService{},
		service.CategoryService{},
		service.AuthService{},
		service.UserService{},
		service.OrderService{},
		service.AddressService{},
		service.SellerService{},
		service.ReviewService{},
		service.BackofficeService{},
		nil,
	)

	var out bytes.Buffer
	if err := handler.templates["profile"].ExecuteTemplate(&out, "layout", map[string]any{
		"User":    &userInfo{UserID: 7, Role: "buyer", FullName: "Buyer"},
		"Profile": model.User{ID: 7, Email: "buyer@example.com", FullName: "Buyer User", Role: model.RoleBuyer, CreatedAt: time.Now()},
	}); err != nil {
		t.Fatalf("render profile template: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`action="/profile/password"`)) {
		t.Fatal("password change form was not rendered")
	}
}

func TestBuildCartDisplayUsesCurrentProductPrice(t *testing.T) {
	ctrl := gomock.NewController(t)
	productRepo := mock_service.NewMockProductRepo(ctrl)

	productRepo.EXPECT().
		GetProductByID(gomock.Any(), int64(10)).
		Return(model.Product{
			ID:    10,
			Name:  "Updated product",
			Price: decimal.NewFromInt(150),
		}, nil)

	handler := &WebHandler{
		productService: service.NewProductService(productRepo, nil, nil, testsupport.PassThroughTxManager{}),
	}
	items := []model.OrderItem{{
		ID:              1,
		ProductID:       10,
		Quantity:        2,
		PriceAtPurchase: decimal.NewFromInt(100),
	}}

	displayItems, total, hasDeletedItems := handler.buildCartDisplay(context.Background(), items)

	if len(displayItems) != 1 {
		t.Fatalf("display item count: got %d, want 1", len(displayItems))
	}
	if !displayItems[0].UnitPrice.Equal(decimal.NewFromInt(150)) {
		t.Fatalf("unit price: got %s, want 150", displayItems[0].UnitPrice)
	}
	if !total.Equal(decimal.NewFromInt(300)) {
		t.Fatalf("cart total: got %s, want 300", total)
	}
	if hasDeletedItems {
		t.Fatal("hasDeletedItems: got true, want false")
	}
}

func TestBuildCartDisplayMarksDeletedProducts(t *testing.T) {
	ctrl := gomock.NewController(t)
	productRepo := mock_service.NewMockProductRepo(ctrl)
	deletedAt := time.Now()

	productRepo.EXPECT().
		GetProductByID(gomock.Any(), int64(10)).
		Return(model.Product{
			ID:        10,
			Name:      "Deleted product",
			Price:     decimal.NewFromInt(150),
			DeletedAt: &deletedAt,
		}, nil)

	handler := &WebHandler{
		productService: service.NewProductService(productRepo, nil, nil, testsupport.PassThroughTxManager{}),
	}
	items := []model.OrderItem{{
		ID:              1,
		ProductID:       10,
		Quantity:        1,
		PriceAtPurchase: decimal.NewFromInt(100),
	}}

	displayItems, _, hasDeletedItems := handler.buildCartDisplay(context.Background(), items)

	if !hasDeletedItems {
		t.Fatal("hasDeletedItems: got false, want true")
	}
	if displayItems[0].DeletedAt == nil {
		t.Fatal("DeletedAt: got nil, want timestamp")
	}
}

func TestBuildOrderItemsDisplayUsesSnapshotPrice(t *testing.T) {
	ctrl := gomock.NewController(t)
	productRepo := mock_service.NewMockProductRepo(ctrl)

	productRepo.EXPECT().
		GetProductByID(gomock.Any(), int64(10)).
		Return(model.Product{
			ID:    10,
			Name:  "Updated product",
			Price: decimal.NewFromInt(150),
		}, nil)

	handler := &WebHandler{
		productService: service.NewProductService(productRepo, nil, nil, testsupport.PassThroughTxManager{}),
	}
	items := []model.OrderItem{{
		ID:              1,
		ProductID:       10,
		Quantity:        2,
		PriceAtPurchase: decimal.NewFromInt(100),
	}}

	displayItems := handler.buildOrderItemsDisplay(context.Background(), items)

	if len(displayItems) != 1 {
		t.Fatalf("display item count: got %d, want 1", len(displayItems))
	}
	if !displayItems[0].UnitPrice.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("unit price: got %s, want 100", displayItems[0].UnitPrice)
	}
	if displayItems[0].TotalPrice != "200" {
		t.Fatalf("total price: got %s, want 200", displayItems[0].TotalPrice)
	}
}
