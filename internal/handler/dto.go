package handler

import (
	"time"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/shopspring/decimal"
)

type UserDTO struct {
	ID        int64          `json:"id"`
	Email     string         `json:"email"`
	FullName  string         `json:"full_name"`
	Phone     *string        `json:"phone"`
	Role      model.UserRole `json:"role"`
	CreatedAt time.Time      `json:"created_at"`
}

func userDTO(user model.User) UserDTO {
	return UserDTO{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}

type SellerDTO struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	CompanyName string    `json:"company_name"`
	Description *string   `json:"description"`
	Rating      *float32  `json:"rating"`
	CreatedAt   time.Time `json:"created_at"`
}

func sellerDTO(s model.Seller) SellerDTO {
	return SellerDTO{
		ID:          s.ID,
		UserID:      s.UserID,
		CompanyName: s.CompanyName,
		Description: s.Description,
		Rating:      s.Rating,
		CreatedAt:   s.CreatedAt,
	}
}

type SellerStatsDTO struct {
	TotalOrders    int64           `json:"total_orders"`
	TotalRevenue   decimal.Decimal `json:"total_revenue"`
	AvgOrderValue  decimal.Decimal `json:"avg_order_value"`
	TopProductName string          `json:"top_product_name"`
}

func sellerStatsDTO(ss model.SellerStats) SellerStatsDTO {
	return SellerStatsDTO{
		TotalOrders:    ss.TotalOrders,
		TotalRevenue:   ss.TotalRevenue,
		AvgOrderValue:  ss.AvgOrderValue,
		TopProductName: ss.TopProductName,
	}
}

type ProductDTO struct {
	ID          int64           `json:"id"`
	SellerID    int64           `json:"seller_id"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Price       decimal.Decimal `json:"price"`
	CreatedAt   time.Time       `json:"created_at"`
	// DeletedAt is omitted for active products and present for soft-deleted
	// ones so clients (e.g. order history) can distinguish them. Public
	// catalog still hides deleted products at the repository layer.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func productDTO(p model.Product) ProductDTO {
	return ProductDTO{
		ID:          p.ID,
		SellerID:    p.SellerID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		CreatedAt:   p.CreatedAt,
		DeletedAt:   p.DeletedAt,
	}
}

type ProductPriceHistoryDTO struct {
	ID        int64           `json:"id"`
	ProductID int64           `json:"product_id"`
	OldPrice  decimal.Decimal `json:"old_price"`
	NewPrice  decimal.Decimal `json:"new_price"`
	ChangedAt time.Time       `json:"changed_at"`
}

func productPriceHistoryDTO(ph model.ProductPriceHistory) ProductPriceHistoryDTO {
	return ProductPriceHistoryDTO{
		ID:        ph.ID,
		ProductID: ph.ProductID,
		OldPrice:  ph.OldPrice,
		NewPrice:  ph.NewPrice,
		ChangedAt: ph.ChangedAt,
	}
}

type CategoryDTO struct {
	ID          int64   `json:"id"`
	ParentID    *int64  `json:"parent_id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func categoryDTO(c model.Category) CategoryDTO {
	return CategoryDTO{
		ID:          c.ID,
		ParentID:    c.ParentID,
		Name:        c.Name,
		Description: c.Description,
	}
}

type AddressDTO struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	City      string    `json:"city"`
	Street    string    `json:"street"`
	House     string    `json:"house"`
	ZipCode   string    `json:"zip_code"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

func addressDTO(a model.Address) AddressDTO {
	return AddressDTO{
		ID:        a.ID,
		UserID:    a.UserID,
		City:      a.City,
		Street:    a.Street,
		House:     a.House,
		ZipCode:   a.ZipCode,
		IsDefault: a.IsDefault,
		CreatedAt: a.CreatedAt,
	}
}

type OrderDTO struct {
	ID          int64           `json:"id"`
	UserID      int64           `json:"user_id"`
	AddressID   *int64          `json:"address_id"`
	SellerID    *int64          `json:"seller_id"`
	Status      string          `json:"status"`
	TotalAmount decimal.Decimal `json:"total_amount"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func orderDTO(o model.Order) OrderDTO {
	return OrderDTO{
		ID:          o.ID,
		UserID:      o.UserID,
		AddressID:   o.AddressID,
		SellerID:    o.SellerID,
		Status:      string(o.Status),
		TotalAmount: o.TotalAmount,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}
}

type OrderItemDTO struct {
	ID              int64           `json:"id"`
	OrderID         int64           `json:"order_id"`
	ProductID       int64           `json:"product_id"`
	Quantity        int             `json:"quantity"`
	PriceAtPurchase decimal.Decimal `json:"price_at_purchase"`
}

func orderItemDTO(oi model.OrderItem) OrderItemDTO {
	return OrderItemDTO{
		ID:              oi.ID,
		OrderID:         oi.OrderID,
		ProductID:       oi.ProductID,
		Quantity:        oi.Quantity,
		PriceAtPurchase: oi.PriceAtPurchase,
	}
}

type PaymentLinkResponse struct {
	OrderID    int64     `json:"order_id"`
	PaymentURL string    `json:"payment_url"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type ReviewDTO struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	ProductID int64     `json:"product_id"`
	Rating    int8      `json:"rating"`
	Comment   *string   `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

func reviewDTO(r model.Review) ReviewDTO {
	return ReviewDTO{
		ID:        r.ID,
		UserID:    r.UserID,
		ProductID: r.ProductID,
		Rating:    r.Rating,
		Comment:   r.Comment,
		CreatedAt: r.CreatedAt,
	}
}
