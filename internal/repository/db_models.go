package repository

import (
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/shopspring/decimal"
)

type userRow struct {
	ID           int64
	Email        string
	PasswordHash string
	FullName     string
	Phone        *string
	Role         m.UserRole
	CreatedAt    time.Time
	DeletedAt    *time.Time
}

func (ur userRow) toModel() m.User {
	return m.User{
		ID:           ur.ID,
		Email:        ur.Email,
		PasswordHash: ur.PasswordHash,
		FullName:     ur.FullName,
		Phone:        ur.Phone,
		Role:         ur.Role,
		CreatedAt:    ur.CreatedAt,
		DeletedAt:    ur.DeletedAt,
	}
}

type addressRow struct {
	ID        int64
	UserID    int64
	City      string
	Street    string
	ZipCode   string
	IsDefault bool
	CreatedAt time.Time
}

func (ar addressRow) toModel() m.Address {
	return m.Address{
		ID:        ar.ID,
		UserID:    ar.UserID,
		City:      ar.City,
		Street:    ar.Street,
		ZipCode:   ar.ZipCode,
		IsDefault: ar.IsDefault,
		CreatedAt: ar.CreatedAt,
	}
}

type sellerRow struct {
	ID          int64
	UserID      int64
	CompanyName string
	Description *string
	Rating      *float32
	CreatedAt   time.Time
}

func (sr sellerRow) toModel() m.Seller {
	return m.Seller{
		ID:          sr.ID,
		UserID:      sr.UserID,
		CompanyName: sr.CompanyName,
		Description: sr.Description,
		Rating:      sr.Rating,
		CreatedAt:   sr.CreatedAt,
	}
}

type sellerStatsRow struct {
	TotalOrders    int64
	TotalRevenue   decimal.Decimal
	AvgOrderValue  decimal.Decimal
	TopProductName string
}

func (sr sellerStatsRow) toModel() m.SellerStats {
	return m.SellerStats{
		TotalOrders:    sr.TotalOrders,
		TotalRevenue:   sr.TotalRevenue,
		AvgOrderValue:  sr.AvgOrderValue,
		TopProductName: sr.TopProductName,
	}
}

type productRow struct {
	ID            int64
	SellerID      int64
	Name          string
	Description   *string
	Price         decimal.Decimal
	StockQuantity int
	CreatedAt     time.Time
	DeletedAt     *time.Time
}

func (pr productRow) toModel() m.Product {
	return m.Product{
		ID:            pr.ID,
		SellerID:      pr.SellerID,
		Name:          pr.Name,
		Description:   pr.Description,
		Price:         pr.Price,
		StockQuantity: pr.StockQuantity,
		CreatedAt:     pr.CreatedAt,
		DeletedAt:     pr.DeletedAt,
	}
}

type productPriceHistoryRow struct {
	ID        int64
	ProductID int64
	OldPrice  decimal.Decimal
	NewPrice  decimal.Decimal
	ChangedAt time.Time
	ChangedBy string
}

func (hr productPriceHistoryRow) toModel() m.ProductPriceHistory {
	return m.ProductPriceHistory{
		ID:        hr.ID,
		ProductID: hr.ProductID,
		OldPrice:  hr.OldPrice,
		NewPrice:  hr.NewPrice,
		ChangedAt: hr.ChangedAt,
		ChangedBy: hr.ChangedBy,
	}
}

type categoryRow struct {
	ID          int64
	ParentID    *int64
	Name        string
	Description *string
}

func (cr categoryRow) toModel() m.Category {
	return m.Category{
		ID:          cr.ID,
		ParentID:    cr.ParentID,
		Name:        cr.Name,
		Description: cr.Description,
	}
}

type reviewRow struct {
	ID        int64
	UserID    int64
	ProductID int64
	Rating    int8
	Comment   *string
	CreatedAt time.Time
}

func (rr reviewRow) toModel() m.Review {
	return m.Review{
		ID:        rr.ID,
		UserID:    rr.UserID,
		ProductID: rr.ProductID,
		Rating:    rr.Rating,
		Comment:   rr.Comment,
		CreatedAt: rr.CreatedAt,
	}
}

type orderItemRow struct {
	ID              int64
	OrderID         int64
	ProductID       int64
	Quantity        int
	PriceAtPurchase decimal.Decimal
}

func (oir orderItemRow) toModel() m.OrderItem {
	return m.OrderItem{
		ID:              oir.ID,
		OrderID:         oir.OrderID,
		ProductID:       oir.ProductID,
		Quantity:        oir.Quantity,
		PriceAtPurchase: oir.PriceAtPurchase,
	}
}

type orderRow struct {
	ID          int64
	UserID      int64
	AddressID   *int64
	SellerID    *int64
	Status      string
	TotalAmount decimal.Decimal
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (or orderRow) toModel() m.Order {
	return m.Order{
		ID:          or.ID,
		UserID:      or.UserID,
		AddressID:   or.AddressID,
		SellerID:    or.SellerID,
		Status:      m.OrderStatus(or.Status),
		TotalAmount: or.TotalAmount,
		CreatedAt:   or.CreatedAt,
		UpdatedAt:   or.UpdatedAt,
	}
}
