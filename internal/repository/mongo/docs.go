package mongorepo

import (
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type userDoc struct {
	ID           int64      `bson:"id"`
	Email        string     `bson:"email"`
	PasswordHash string     `bson:"password_hash"`
	FullName     string     `bson:"full_name"`
	Phone        *string    `bson:"phone,omitempty"`
	Role         m.UserRole `bson:"role"`
	CreatedAt    time.Time  `bson:"created_at"`
	DeletedAt    *time.Time `bson:"deleted_at,omitempty"`
}

func (d userDoc) toModel() m.User {
	return m.User{
		ID:           d.ID,
		Email:        d.Email,
		PasswordHash: d.PasswordHash,
		FullName:     d.FullName,
		Phone:        d.Phone,
		Role:         d.Role,
		CreatedAt:    d.CreatedAt,
		DeletedAt:    d.DeletedAt,
	}
}

type sellerDoc struct {
	ID          int64      `bson:"id"`
	UserID      int64      `bson:"user_id"`
	CompanyName string     `bson:"company_name"`
	Description *string    `bson:"description,omitempty"`
	Rating      *float32   `bson:"rating,omitempty"`
	CreatedAt   time.Time  `bson:"created_at"`
	DeletedAt   *time.Time `bson:"deleted_at,omitempty"`
}

func (d sellerDoc) toModel() m.Seller {
	return m.Seller{
		ID:          d.ID,
		UserID:      d.UserID,
		CompanyName: d.CompanyName,
		Description: d.Description,
		Rating:      d.Rating,
		CreatedAt:   d.CreatedAt,
	}
}

type categoryDoc struct {
	ID          int64   `bson:"id"`
	ParentID    *int64  `bson:"parent_id,omitempty"`
	Name        string  `bson:"name"`
	Description *string `bson:"description,omitempty"`
}

func (d categoryDoc) toModel() m.Category {
	return m.Category{
		ID:          d.ID,
		ParentID:    d.ParentID,
		Name:        d.Name,
		Description: d.Description,
	}
}

type productDoc struct {
	ID               int64           `bson:"id"`
	SellerID         int64           `bson:"seller_id"`
	Name             string          `bson:"name"`
	Description      *string         `bson:"description,omitempty"`
	Price            bson.Decimal128 `bson:"price"`
	StockQuantity    int             `bson:"stock_quantity"`
	ReservedQuantity int             `bson:"reserved_quantity"`
	Rating           *float32        `bson:"rating,omitempty"`
	CreatedAt        time.Time       `bson:"created_at"`
	DeletedAt        *time.Time      `bson:"deleted_at,omitempty"`
	LockVersion      int64           `bson:"lock_version"`
}

func newProductDoc(id int64, pc m.ProductCreate) (productDoc, error) {
	price, err := decimalToBSON(pc.Price)
	if err != nil {
		return productDoc{}, err
	}
	return productDoc{
		ID:            id,
		SellerID:      pc.SellerID,
		Name:          pc.Name,
		Description:   pc.Description,
		Price:         price,
		StockQuantity: pc.StockQuantity,
		CreatedAt:     nowUTC(),
	}, nil
}

func (d productDoc) toModel() (m.Product, error) {
	price, err := decimalFromBSON(d.Price)
	if err != nil {
		return m.Product{}, err
	}
	return m.Product{
		ID:               d.ID,
		SellerID:         d.SellerID,
		Name:             d.Name,
		Description:      d.Description,
		Price:            price,
		StockQuantity:    d.StockQuantity,
		ReservedQuantity: d.ReservedQuantity,
		Rating:           d.Rating,
		CreatedAt:        d.CreatedAt,
		DeletedAt:        d.DeletedAt,
	}, nil
}

type productCategoryDoc struct {
	ProductID  int64 `bson:"product_id"`
	CategoryID int64 `bson:"category_id"`
}

type addressDoc struct {
	ID        int64     `bson:"id"`
	UserID    int64     `bson:"user_id"`
	City      string    `bson:"city"`
	Street    string    `bson:"street"`
	House     string    `bson:"house"`
	ZipCode   string    `bson:"zip_code"`
	IsDefault bool      `bson:"is_default"`
	CreatedAt time.Time `bson:"created_at"`
}

func (d addressDoc) toModel() m.Address {
	return m.Address{
		ID:        d.ID,
		UserID:    d.UserID,
		City:      d.City,
		Street:    d.Street,
		House:     d.House,
		ZipCode:   d.ZipCode,
		IsDefault: d.IsDefault,
		CreatedAt: d.CreatedAt,
	}
}

type orderDoc struct {
	ID          int64           `bson:"id"`
	UserID      int64           `bson:"user_id"`
	AddressID   *int64          `bson:"address_id,omitempty"`
	SellerID    *int64          `bson:"seller_id,omitempty"`
	Status      m.OrderStatus   `bson:"status"`
	TotalAmount bson.Decimal128 `bson:"total_amount"`
	CreatedAt   time.Time       `bson:"created_at"`
	UpdatedAt   time.Time       `bson:"updated_at"`
}

func newOrderDoc(id int64, oc m.OrderCreate) (orderDoc, error) {
	total, err := decimalToBSON(oc.TotalAmount)
	if err != nil {
		return orderDoc{}, err
	}
	now := nowUTC()
	return orderDoc{
		ID:          id,
		UserID:      oc.UserID,
		AddressID:   oc.AddressID,
		SellerID:    oc.SellerID,
		Status:      oc.Status,
		TotalAmount: total,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (d orderDoc) toModel() (m.Order, error) {
	total, err := decimalFromBSON(d.TotalAmount)
	if err != nil {
		return m.Order{}, err
	}
	return m.Order{
		ID:          d.ID,
		UserID:      d.UserID,
		AddressID:   d.AddressID,
		SellerID:    d.SellerID,
		Status:      d.Status,
		TotalAmount: total,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}, nil
}

type orderItemDoc struct {
	ID              int64           `bson:"id"`
	OrderID         int64           `bson:"order_id"`
	ProductID       int64           `bson:"product_id"`
	Quantity        int             `bson:"quantity"`
	PriceAtPurchase bson.Decimal128 `bson:"price_at_purchase"`
}

func newOrderItemDoc(id int64, oic m.OrderItemCreate) (orderItemDoc, error) {
	price, err := decimalToBSON(oic.PriceAtPurchase)
	if err != nil {
		return orderItemDoc{}, err
	}
	return orderItemDoc{
		ID:              id,
		OrderID:         oic.OrderID,
		ProductID:       oic.ProductID,
		Quantity:        oic.Quantity,
		PriceAtPurchase: price,
	}, nil
}

func (d orderItemDoc) toModel() (m.OrderItem, error) {
	price, err := decimalFromBSON(d.PriceAtPurchase)
	if err != nil {
		return m.OrderItem{}, err
	}
	return m.OrderItem{
		ID:              d.ID,
		OrderID:         d.OrderID,
		ProductID:       d.ProductID,
		Quantity:        d.Quantity,
		PriceAtPurchase: price,
	}, nil
}

type reviewDoc struct {
	ID        int64     `bson:"id"`
	UserID    int64     `bson:"user_id"`
	ProductID int64     `bson:"product_id"`
	Rating    int8      `bson:"rating"`
	Comment   *string   `bson:"comment,omitempty"`
	CreatedAt time.Time `bson:"created_at"`
}

func (d reviewDoc) toModel() m.Review {
	return m.Review{
		ID:        d.ID,
		UserID:    d.UserID,
		ProductID: d.ProductID,
		Rating:    d.Rating,
		Comment:   d.Comment,
		CreatedAt: d.CreatedAt,
	}
}

type priceHistoryDoc struct {
	ID        int64           `bson:"id"`
	ProductID int64           `bson:"product_id"`
	OldPrice  bson.Decimal128 `bson:"old_price"`
	NewPrice  bson.Decimal128 `bson:"new_price"`
	ChangedAt time.Time       `bson:"changed_at"`
	ChangedBy string          `bson:"changed_by"`
}

func newPriceHistoryDoc(id int64, productID int64, oldPrice decimal.Decimal, newPrice decimal.Decimal, changedBy string) (priceHistoryDoc, error) {
	oldBSON, err := decimalToBSON(oldPrice)
	if err != nil {
		return priceHistoryDoc{}, err
	}
	newBSON, err := decimalToBSON(newPrice)
	if err != nil {
		return priceHistoryDoc{}, err
	}
	return priceHistoryDoc{
		ID:        id,
		ProductID: productID,
		OldPrice:  oldBSON,
		NewPrice:  newBSON,
		ChangedAt: nowUTC(),
		ChangedBy: changedBy,
	}, nil
}

func (d priceHistoryDoc) toModel() (m.ProductPriceHistory, error) {
	oldPrice, err := decimalFromBSON(d.OldPrice)
	if err != nil {
		return m.ProductPriceHistory{}, err
	}
	newPrice, err := decimalFromBSON(d.NewPrice)
	if err != nil {
		return m.ProductPriceHistory{}, err
	}
	return m.ProductPriceHistory{
		ID:        d.ID,
		ProductID: d.ProductID,
		OldPrice:  oldPrice,
		NewPrice:  newPrice,
		ChangedAt: d.ChangedAt,
		ChangedBy: d.ChangedBy,
	}, nil
}

type favoriteDoc struct {
	UserID    int64     `bson:"user_id"`
	ProductID int64     `bson:"product_id"`
	CreatedAt time.Time `bson:"created_at"`
}
