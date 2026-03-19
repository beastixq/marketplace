package handler

import (
	"time"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/shopspring/decimal"
)

type User struct {
	ID        int64          `json:"id"`
	Email     string         `json:"email"`
	FullName  string         `json:"full_name"`
	Phone     *string        `json:"phone"`
	Role      model.UserRole `json:"role"`
	CreatedAt time.Time      `json:"created_at"`
}

func userFromService(user model.User) User {
	return User{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}

type Seller struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	CompanyName string    `json:"company_name"`
	Description *string   `json:"description"`
	Rating      *float32  `json:"rating"`
	CreatedAt   time.Time `json:"created_at"`
}

func sellerFromService(s model.Seller) Seller {
	return Seller{
		ID:          s.ID,
		UserID:      s.UserID,
		CompanyName: s.CompanyName,
		Description: s.Description,
		Rating:      s.Rating,
		CreatedAt:   s.CreatedAt,
	}
}

type SellerStats struct {
	TotalOrders    int64           `json:"total_orders"`
	TotalRevenue   decimal.Decimal `json:"total_revenue"`
	AvgOrderValue  decimal.Decimal `json:"avg_order_value"`
	TopProductName string          `json:"top_product_name"`
}

func sellerStatsFromService(ss model.SellerStats) SellerStats {
	return SellerStats{
		TotalOrders:    ss.TotalOrders,
		TotalRevenue:   ss.TotalRevenue,
		AvgOrderValue:  ss.AvgOrderValue,
		TopProductName: ss.TopProductName,
	}
}

type Product struct {
	ID          int64           `json:"id"`
	SellerID    int64           `json:"seller_id"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Price       decimal.Decimal `json:"price"`
	CreatedAt   time.Time       `json:"created_at"`
}

func productFromService(p model.Product) Product {
	return Product{
		ID:          p.ID,
		SellerID:    p.SellerID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		CreatedAt:   p.CreatedAt,
	}
}

type ProductPriceHistory struct {
	ID        int64           `json:"id"`
	ProductID int64           `json:"product_id"`
	OldPrice  decimal.Decimal `json:"old_price"`
	NewPrice  decimal.Decimal `json:"new_price"`
	ChangedAt time.Time       `json:"changed_at"`
}

func productPriceHistoryFromService(ph model.ProductPriceHistory) ProductPriceHistory {
	return ProductPriceHistory{
		ID:        ph.ID,
		ProductID: ph.ProductID,
		OldPrice:  ph.OldPrice,
		NewPrice:  ph.NewPrice,
		ChangedAt: ph.ChangedAt,
	}
}

type Category struct {
	ID          int64   `json:"id"`
	ParentID    *int64  `json:"parent_id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func categoryFromService(c model.Category) Category {
	return Category{ID: c.ID, ParentID: c.ParentID, Name: c.Name, Description: c.Description}
}

type Address struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	City      string    `json:"city"`
	Street    string    `json:"street"`
	ZipCode   string    `json:"zip_code"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

func addressFromService(a model.Address) Address {
	return Address{ID: a.ID, UserID: a.UserID, City: a.City, Street: a.Street, ZipCode: a.ZipCode, IsDefault: a.IsDefault, CreatedAt: a.CreatedAt}
}

type Order struct {
	ID          int64           `json:"id"`
	UserID      int64           `json:"user_id"`
	AddressID   *int64          `json:"address_id"`
	SellerID    *int64          `json:"seller_id"`
	Status      string          `json:"status"`
	TotalAmount decimal.Decimal `json:"total_amount"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func orderFromService(o model.Order) Order {
	return Order{ID: o.ID, UserID: o.UserID, AddressID: o.AddressID, SellerID: o.SellerID, Status: string(o.Status), TotalAmount: o.TotalAmount, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}
}

type OrderItem struct {
	ID              int64           `json:"id"`
	OrderID         int64           `json:"order_id"`
	ProductID       int64           `json:"product_id"`
	Quantity        int             `json:"quantity"`
	PriceAtPurchase decimal.Decimal `json:"price_at_purchase"`
}

func orderItemFromService(oi model.OrderItem) OrderItem {
	return OrderItem{ID: oi.ID, OrderID: oi.OrderID, ProductID: oi.ProductID, Quantity: oi.Quantity, PriceAtPurchase: oi.PriceAtPurchase}
}

type Review struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	ProductID int64     `json:"product_id"`
	Rating    int8      `json:"rating"`
	Comment   *string   `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

func reviewFromService(r model.Review) Review {
	return Review{
		ID:        r.ID,
		UserID:    r.UserID,
		ProductID: r.ProductID,
		Rating:    r.Rating,
		Comment:   r.Comment,
		CreatedAt: r.CreatedAt,
	}
}
