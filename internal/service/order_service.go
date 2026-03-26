package service

import (
	"context"
	"errors"
	"fmt"

	m "github.com/beastixq/marketplace/internal/model"

	"github.com/shopspring/decimal"
)

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_order_item_repo.go github.com/beastixq/marketplace/internal/service OrderItemRepo
type OrderItemRepo interface {
	GetOrderItemByID(ctx context.Context, id int64) (oi m.OrderItem, err error)
	GetOrderItemsByOrderID(ctx context.Context, orderID int64) (ois []m.OrderItem, err error)
	CreateOrderItem(ctx context.Context, oic m.OrderItemCreate) (id int64, err error)
	UpdateOrderItem(ctx context.Context, id int64, oiu m.OrderItemUpdate) (oi m.OrderItem, err error)
	DeleteOrderItemByID(ctx context.Context, id int64) (err error)
}

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_order_repo.go github.com/beastixq/marketplace/internal/service OrderRepo
type OrderRepo interface {
	GetOrderByID(ctx context.Context, id int64) (o m.Order, err error)
	GetOrdersByUserID(ctx context.Context, userID int64) (orders []m.Order, err error)
	GetSellerOrdersBySellerID(ctx context.Context, sellerID int64, pg m.PaginationOpts) (orders []m.Order, err error)
	CreateOrder(ctx context.Context, oc m.OrderCreate) (id int64, err error)
	UpdateOrder(ctx context.Context, id int64, ou m.OrderUpdate) (o m.Order, err error)
	DeleteOrderByID(ctx context.Context, id int64) (err error)
}

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_product_getter.go github.com/beastixq/marketplace/internal/service ProductGetter
type ProductGetter interface {
	GetProductByID(ctx context.Context, id int64) (p m.Product, err error)
}

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_seller_getter.go github.com/beastixq/marketplace/internal/service SellerGetter
type SellerGetter interface {
	GetSellerByUserID(ctx context.Context, userID int64) (s m.Seller, err error)
}

type OrderService struct {
	orderRepo     OrderRepo
	orderItemRepo OrderItemRepo
	productGetter ProductGetter
	sellerGetter  SellerGetter
}

func NewOrderService(or OrderRepo, oir OrderItemRepo, pr ProductGetter, sr SellerGetter) OrderService {
	return OrderService{orderRepo: or, orderItemRepo: oir, productGetter: pr, sellerGetter: sr}
}

func (os OrderService) GetOrderByID(ctx context.Context, orderID int64) (order m.Order, err error) {
	order, err = os.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Order{}, ErrOrderNotFound
		}
		return m.Order{}, fmt.Errorf("%w: %v", ErrGetOrderByID, err)
	}
	return order, nil
}

func (os OrderService) GetOrdersByUserID(ctx context.Context, userID int64) (orders []m.Order, err error) {
	orders, err = os.orderRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetOrdersByUserID, err)
	}
	return orders, nil
}

func (os OrderService) GetSellerOrdersByUserID(ctx context.Context, userID int64, pg m.PaginationOpts) (orders []m.Order, err error) {
	seller, err := os.sellerGetter.GetSellerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrSellerNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrGetSellerByUserID, err)
	}

	orders, err = os.orderRepo.GetSellerOrdersBySellerID(ctx, seller.ID, pg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetOrdersBySellerID, err)
	}
	return orders, nil
}

func (os OrderService) GetOrderItemsByOrderID(ctx context.Context, orderID int64) (ois []m.OrderItem, err error) {
	ois, err = os.orderItemRepo.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetOrderItemsByOrderID, err)
	}
	return ois, nil
}

func (os OrderService) GetCart(ctx context.Context, userID int64) (order m.Order, err error) {
	orders, err := os.orderRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return m.Order{}, fmt.Errorf("%w: %v", ErrGetCart, err)
	}
	if len(orders) == 0 {
		return m.Order{}, ErrCartNotFound
	}
	drafts := make([]int, 0)
	for i, ord := range orders {
		if ord.Status == m.StatusDraft {
			drafts = append(drafts, i)
		}
	}
	if len(drafts) == 0 {
		return m.Order{}, ErrCartNotFound
	}
	latest := drafts[0]
	for _, idx := range drafts[1:] {
		if orders[idx].CreatedAt.Compare(orders[latest].CreatedAt) > 0 {
			latest = idx
		}
	}

	cart := orders[latest]
	items, err := os.orderItemRepo.GetOrderItemsByOrderID(ctx, cart.ID)
	if err != nil {
		return m.Order{}, fmt.Errorf("%w: %v", ErrGetCart, err)
	}
	total := decimal.Zero
	for _, item := range items {
		total = total.Add(item.PriceAtPurchase.Mul(decimal.NewFromInt(int64(item.Quantity))))
	}
	cart.TotalAmount = total

	return cart, nil
}

func (os OrderService) AddItemToCart(ctx context.Context, userID int64, productID int64, quantity int) (err error) {
	var cartOrderID int64
	cartOrder, err := os.GetCart(ctx, userID)
	if err != nil {
		if !errors.Is(err, ErrCartNotFound) {
			return fmt.Errorf("%w: %v", ErrGetCart, err)
		}
		cartOrderID, err = os.orderRepo.CreateOrder(ctx, m.OrderCreate{
			UserID:      userID,
			Status:      m.StatusDraft,
			TotalAmount: decimal.Zero,
		})
		if err != nil {
			return fmt.Errorf("%w: %v", ErrCreateOrder, err)
		}
	}
	if cartOrderID == 0 {
		cartOrderID = cartOrder.ID
	}

	existingItems, err := os.orderItemRepo.GetOrderItemsByOrderID(ctx, cartOrderID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrGetOrderItemsByOrderID, err)
	}
	for _, item := range existingItems {
		if item.ProductID == productID {
			return ErrProductAlreadyInCart
		}
	}

	product, err := os.productGetter.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrProductNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}
	if product.StockQuantity < quantity {
		return ErrQuantityTooBig
	}

	_, err = os.orderItemRepo.CreateOrderItem(ctx, m.OrderItemCreate{
		OrderID:         cartOrderID,
		ProductID:       productID,
		Quantity:        quantity,
		PriceAtPurchase: product.Price,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCreateOrderItem, err)
	}
	return nil
}

func (os OrderService) ChangeQuantityCartItem(ctx context.Context, itemID int64, quantity int) (err error) {
	orderItem, err := os.orderItemRepo.GetOrderItemByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrOrderItemNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetOrderItemByID, err)
	}
	product, err := os.productGetter.GetProductByID(ctx, orderItem.ProductID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrProductNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}
	if quantity > product.StockQuantity {
		return ErrQuantityTooBig
	}
	_, err = os.orderItemRepo.UpdateOrderItem(ctx, itemID, m.OrderItemUpdate{Quantity: &quantity})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpdateOrderItem, err)
	}
	return nil
}

func (os OrderService) DeleteCartItem(ctx context.Context, itemID int64) (err error) {
	err = os.orderItemRepo.DeleteOrderItemByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteOrderItemByID, err)
	}
	return nil
}

// TODO: 1. N+1 products read -> less reads
// 2. think about price at purchase. it item was added long ago and now it
// has new price -> create with updated
// 3. delete draft order items before actual order delete (on delete restrict)
// 3. or think about cascade delete
func (os OrderService) Checkout(ctx context.Context, userID int64, addressID int64) (orderIDs []int64, err error) {
	cart, err := os.GetCart(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrCartNotFound) {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrGetCart, err)
	}

	items, err := os.orderItemRepo.GetOrderItemsByOrderID(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetOrderItemsByOrderID, err)
	}
	if len(items) == 0 {
		return nil, ErrEmptyCart
	}

	itemsBySellerIDs := make(map[int64][]m.OrderItem)
	for _, item := range items {
		product, err := os.productGetter.GetProductByID(ctx, item.ProductID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrProductNotFound
			}
			return nil, fmt.Errorf("%w: %v", ErrGetProductByID, err)
		}
		itemsBySellerIDs[product.SellerID] = append(itemsBySellerIDs[product.SellerID], item)
	}

	createdOrdersIDs := make([]int64, 0, len(itemsBySellerIDs))
	for sellerID, sellerItems := range itemsBySellerIDs {
		totalAmount := decimal.Zero
		for _, item := range sellerItems {
			totalAmount = totalAmount.Add(item.PriceAtPurchase.Mul(decimal.NewFromInt(int64(item.Quantity))))
		}

		orderID, err := os.orderRepo.CreateOrder(ctx, m.OrderCreate{
			UserID:      userID,
			AddressID:   &addressID,
			SellerID:    &sellerID,
			Status:      m.StatusPending,
			TotalAmount: totalAmount,
		})
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCreateOrder, err)
		}

		for _, item := range sellerItems {
			_, err = os.orderItemRepo.CreateOrderItem(ctx, m.OrderItemCreate{
				OrderID:         orderID,
				ProductID:       item.ProductID,
				Quantity:        item.Quantity,
				PriceAtPurchase: item.PriceAtPurchase,
			})
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrCreateOrderItem, err)
			}
		}

		createdOrdersIDs = append(createdOrdersIDs, orderID)
	}

	if err = os.orderRepo.DeleteOrderByID(ctx, cart.ID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDeleteOrderByID, err)
	}

	return createdOrdersIDs, nil
}

func (os OrderService) PayOrder(ctx context.Context, orderID int64, userID int64) (err error) {
	order, err := os.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: %v", ErrOrderNotFound, err)
		}
		return fmt.Errorf("%w: %v", ErrGetOrderByID, err)
	}
	if userID != order.UserID {
		return ErrNotYourOrder
	}
	if order.Status != m.StatusPending {
		return fmt.Errorf("%w: status must be 'pending' to make pay", ErrOrderStatusInvalid)
	}
	statusPaid := m.StatusPaid
	_, err = os.orderRepo.UpdateOrder(ctx, orderID, m.OrderUpdate{Status: &statusPaid})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
	}
	return nil
}

func (os OrderService) CancelOrder(ctx context.Context, orderID int64, userID int64) (err error) {
	order, err := os.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: %v", ErrOrderNotFound, err)
		}
		return fmt.Errorf("%w: %v", ErrGetOrderByID, err)
	}
	if userID != order.UserID {
		return ErrNotYourOrder
	}
	if order.Status != m.StatusPending && order.Status != m.StatusPaid {
		return fmt.Errorf("%w: can only cancel pending or paid orders", ErrOrderStatusInvalid)
	}
	statusCancelled := m.StatusCancelled
	_, err = os.orderRepo.UpdateOrder(ctx, orderID, m.OrderUpdate{Status: &statusCancelled})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
	}
	return nil
}

func (os OrderService) ShipOrder(ctx context.Context, orderID int64, userID int64) (err error) {
	seller, err := os.sellerGetter.GetSellerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrSellerNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetSellerByUserID, err)
	}
	order, err := os.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: %v", ErrOrderNotFound, err)
		}
		return fmt.Errorf("%w: %v", ErrGetOrderByID, err)
	}
	if order.SellerID == nil || *order.SellerID != seller.ID {
		return ErrNotYourOrder
	}
	if order.Status != m.StatusPaid {
		return fmt.Errorf("%w: order must be paid before shipping", ErrOrderStatusInvalid)
	}
	statusShipped := m.StatusShipped
	_, err = os.orderRepo.UpdateOrder(ctx, orderID, m.OrderUpdate{Status: &statusShipped})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
	}
	return nil
}

func (os OrderService) DeliverOrder(ctx context.Context, orderID int64, userID int64) (err error) {
	seller, err := os.sellerGetter.GetSellerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrSellerNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetSellerByUserID, err)
	}
	order, err := os.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: %v", ErrOrderNotFound, err)
		}
		return fmt.Errorf("%w: %v", ErrGetOrderByID, err)
	}
	if order.SellerID == nil || *order.SellerID != seller.ID {
		return ErrNotYourOrder
	}
	if order.Status != m.StatusShipped {
		return fmt.Errorf("%w: order must be shipped before delivering", ErrOrderStatusInvalid)
	}
	statusDelivered := m.StatusDelivered
	_, err = os.orderRepo.UpdateOrder(ctx, orderID, m.OrderUpdate{Status: &statusDelivered})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
	}
	return nil
}
