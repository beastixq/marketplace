package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	m "github.com/beastixq/marketplace/internal/model"

	"github.com/shopspring/decimal"
)

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_order_item_repo.go github.com/beastixq/marketplace/internal/service OrderItemRepo
type OrderItemRepo interface {
	GetOrderItemByID(ctx context.Context, id int64) (oi m.OrderItem, err error)
	GetOrderItemsByOrderID(ctx context.Context, orderID int64) (ois []m.OrderItem, err error)
	CreateOrderItem(ctx context.Context, oic m.OrderItemCreate) (id int64, err error)
	UpdateOrderItem(ctx context.Context, id int64, oiu m.OrderItemUpdate) (oi m.OrderItem, err error)
	UpdateOrderItemQtyIfDraft(ctx context.Context, id int64, userID int64, qty int) error
	DeleteOrderItemByID(ctx context.Context, id int64) (err error)
	DeleteOrderItemIfDraft(ctx context.Context, id int64, userID int64) error
}

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_order_repo.go github.com/beastixq/marketplace/internal/service OrderRepo
type OrderRepo interface {
	GetOrderByID(ctx context.Context, id int64) (o m.Order, err error)
	GetOrdersByUserID(ctx context.Context, userID int64, pg m.PaginationOpts) (orders []m.Order, err error)
	GetSellerOrdersBySellerID(ctx context.Context, sellerID int64, pg m.PaginationOpts) (orders []m.Order, err error)
	GetExpiredPendingOrders(ctx context.Context, deadline time.Time) (orders []m.Order, err error)
	CreateOrder(ctx context.Context, oc m.OrderCreate) (id int64, err error)
	UpdateOrder(ctx context.Context, id int64, ou m.OrderUpdate) (o m.Order, err error)
	// UpdateOrderStatus atomically transitions status only if current status is in `from`.
	// Returns ErrNotFound if no row matched (concurrent transition already happened).
	UpdateOrderStatus(ctx context.Context, id int64, from []m.OrderStatus, to m.OrderStatus) error
	// LockUserCart acquires a transaction-scoped advisory lock that serializes
	// cart mutations and Checkout for one user. Caller must be inside a tx.
	LockUserCart(ctx context.Context, userID int64) error
	DeleteOrderByID(ctx context.Context, id int64) (err error)
}

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_seller_getter.go github.com/beastixq/marketplace/internal/service SellerGetter
type SellerGetter interface {
	GetSellerByUserID(ctx context.Context, userID int64) (s m.Seller, err error)
}

type OrderAddressRepo interface {
	GetAddressByID(ctx context.Context, id int64) (a m.Address, err error)
}

type OrderService struct {
	orderRepo     OrderRepo
	orderItemRepo OrderItemRepo
	productRepo   ProductRepo
	addressRepo   OrderAddressRepo
	sellerGetter  SellerGetter
	txManager     TxManager
}

func NewOrderService(or OrderRepo, oir OrderItemRepo, pr ProductRepo, ar OrderAddressRepo, sr SellerGetter, tx TxManager) OrderService {
	return OrderService{
		orderRepo:     or,
		orderItemRepo: oir,
		productRepo:   pr,
		addressRepo:   ar,
		sellerGetter:  sr,
		txManager:     tx}
}

func (os OrderService) GetOrderByID(ctx context.Context, actor Actor, orderID int64) (order m.Order, err error) {
	if !actor.HasRole(m.RoleBuyer, m.RoleSeller, m.RoleAdmin) {
		return m.Order{}, ErrPermissionDenied
	}

	order, err = os.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Order{}, ErrOrderNotFound
		}
		return m.Order{}, fmt.Errorf("%w: %v", ErrGetOrderByID, err)
	}
	if actor.IsAdmin() {
		return order, nil
	}

	switch actor.Role {
	case m.RoleBuyer:
		if order.UserID != actor.UserID {
			return m.Order{}, ErrNotYourOrder
		}
	case m.RoleSeller:
		seller, err := os.sellerGetter.GetSellerByUserID(ctx, actor.UserID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return m.Order{}, ErrSellerNotFound
			}
			return m.Order{}, fmt.Errorf("%w: %v", ErrGetSellerByUserID, err)
		}
		if order.SellerID == nil || *order.SellerID != seller.ID {
			return m.Order{}, ErrNotYourOrder
		}
	default:
		return m.Order{}, ErrPermissionDenied
	}
	return order, nil
}

func (os OrderService) GetOrdersByUserID(ctx context.Context, actor Actor, pg m.PaginationOpts) (orders []m.Order, err error) {
	if !actor.HasRole(m.RoleBuyer) {
		return nil, ErrPermissionDenied
	}

	orders, err = os.orderRepo.GetOrdersByUserID(ctx, actor.UserID, pg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetOrdersByUserID, err)
	}
	return orders, nil
}

func (os OrderService) GetSellerOrdersByUserID(ctx context.Context, actor Actor, pg m.PaginationOpts) (orders []m.Order, err error) {
	if !actor.HasRole(m.RoleSeller) {
		return nil, ErrPermissionDenied
	}

	seller, err := os.sellerGetter.GetSellerByUserID(ctx, actor.UserID)
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

func (os OrderService) GetOrderItemsByOrderID(ctx context.Context, actor Actor, orderID int64) (ois []m.OrderItem, err error) {
	if _, err = os.GetOrderByID(ctx, actor, orderID); err != nil {
		return nil, err
	}
	ois, err = os.orderItemRepo.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetOrderItemsByOrderID, err)
	}
	return ois, nil
}

func (os OrderService) GetCart(ctx context.Context, actor Actor) (order m.Order, err error) {
	if !actor.HasRole(m.RoleBuyer) {
		return m.Order{}, ErrPermissionDenied
	}

	orders, err := os.orderRepo.GetOrdersByUserID(ctx, actor.UserID, m.PaginationOpts{})
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
	// defensive code: если вдруг по какой-то причине корзин в базе >0, возвращаем последнюю созданную
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

func (os OrderService) AddItemToCart(ctx context.Context, actor Actor, productID int64, quantity int) error {
	if !actor.HasRole(m.RoleBuyer) {
		return ErrPermissionDenied
	}

	return os.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Advisory lock serializes cart mutations and Checkout for this user.
		if err := os.orderRepo.LockUserCart(ctx, actor.UserID); err != nil {
			return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
		}

		var cartOrderID int64
		cartOrder, err := os.GetCart(ctx, actor)
		if err != nil && !errors.Is(err, ErrCartNotFound) {
			return fmt.Errorf("%w: %v", ErrGetCart, err)
		}
		if err == nil {
			cartOrderID = cartOrder.ID
			existingItems, err := os.orderItemRepo.GetOrderItemsByOrderID(ctx, cartOrderID)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrGetOrderItemsByOrderID, err)
			}
			for _, item := range existingItems {
				if item.ProductID == productID {
					return ErrProductAlreadyInCart
				}
			}
		}

		product, err := os.productRepo.GetProductByID(ctx, productID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrProductNotFound
			}
			return fmt.Errorf("%w: %v", ErrGetProductByID, err)
		}
		if product.DeletedAt != nil {
			return ErrProductDeleted
		}
		if product.AvailableQuantity() < quantity {
			return ErrQuantityTooBig
		}

		if cartOrderID == 0 {
			cartOrderID, err = os.orderRepo.CreateOrder(ctx, m.OrderCreate{
				UserID:      actor.UserID,
				Status:      m.StatusDraft,
				TotalAmount: decimal.Zero,
			})
			if err != nil {
				return fmt.Errorf("%w: %v", ErrCreateOrder, err)
			}
		}
		_, err = os.orderItemRepo.CreateOrderItem(ctx, m.OrderItemCreate{
			OrderID:         cartOrderID,
			ProductID:       productID,
			Quantity:        quantity,
			PriceAtPurchase: product.Price,
		})
		if err != nil {
			if errors.Is(err, ErrProductAlreadyInCart) {
				return ErrProductAlreadyInCart
			}
			return fmt.Errorf("%w: %v", ErrCreateOrderItem, err)
		}
		return nil
	})
}

func (os OrderService) ChangeQuantityCartItem(ctx context.Context, actor Actor, itemID int64, quantity int) (err error) {
	if !actor.HasRole(m.RoleBuyer) {
		return ErrPermissionDenied
	}

	return os.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		if err := os.orderRepo.LockUserCart(ctx, actor.UserID); err != nil {
			return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
		}

		orderItem, err := os.orderItemRepo.GetOrderItemByID(ctx, itemID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrOrderItemNotFound
			}
			return fmt.Errorf("%w: %v", ErrGetOrderItemByID, err)
		}
		product, err := os.productRepo.GetProductByID(ctx, orderItem.ProductID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrProductNotFound
			}
			return fmt.Errorf("%w: %v", ErrGetProductByID, err)
		}
		if quantity > product.AvailableQuantity() {
			return ErrQuantityTooBig
		}

		// Conditional UPDATE atomically enforces ownership + status=draft;
		// 0 rows affected means another tx claimed the cart or it isn't this user's.
		if err := os.orderItemRepo.UpdateOrderItemQtyIfDraft(ctx, itemID, actor.UserID, quantity); err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrOrderStatusInvalid
			}
			return fmt.Errorf("%w: %v", ErrUpdateOrderItem, err)
		}
		return nil
	})
}

func (os OrderService) DeleteCartItem(ctx context.Context, actor Actor, itemID int64) (err error) {
	if !actor.HasRole(m.RoleBuyer) {
		return ErrPermissionDenied
	}

	return os.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		if err := os.orderRepo.LockUserCart(ctx, actor.UserID); err != nil {
			return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
		}
		if err := os.orderItemRepo.DeleteOrderItemIfDraft(ctx, itemID, actor.UserID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrOrderItemNotFound
			}
			return fmt.Errorf("%w: %v", ErrDeleteOrderItemByID, err)
		}
		return nil
	})
}

func (os OrderService) Checkout(ctx context.Context, actor Actor, addressID int64) (orderIDs []int64, err error) {
	if !actor.HasRole(m.RoleBuyer) {
		return nil, ErrPermissionDenied
	}

	var createdOrdersIDs []int64
	err = os.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Advisory lock serializes Checkout with parallel cart mutations of the same user.
		if err := os.orderRepo.LockUserCart(ctx, actor.UserID); err != nil {
			return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
		}

		address, err := os.addressRepo.GetAddressByID(ctx, addressID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrAddressNotFound
			}
			return fmt.Errorf("%w: %v", ErrGetAddressByID, err)
		}
		if address.UserID != actor.UserID {
			return ErrNotYourAddress
		}

		cart, err := os.GetCart(ctx, actor)
		if err != nil {
			if errors.Is(err, ErrCartNotFound) {
				return ErrCartNotFound
			}
			return fmt.Errorf("%w: %v", ErrGetCart, err)
		}

		// claim cart — atomic draft→pending prevents concurrent double-checkout;
		// 0 rows affected means another tx already claimed it
		checkoutFrom, checkoutTo := orderTransitionStatuses(orderTransitionCheckout)
		if err := os.orderRepo.UpdateOrderStatus(ctx, cart.ID, checkoutFrom, checkoutTo); err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrCartNotFound
			}
			return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
		}

		// re-read items inside tx — consistent with claim, no stale add/remove races
		items, err := os.orderItemRepo.GetOrderItemsByOrderID(ctx, cart.ID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrGetOrderItemsByOrderID, err)
		}
		if len(items) == 0 {
			return ErrEmptyCart
		}

		// суммируем qty по productID и строим отсортированный список ID —
		// детерминированный порядок блокировок предотвращает deadlock
		// (T1: lock 1→2, T2: lock 1→2 — очередь; без сортировки T1: lock 1, T2: lock 2 → deadlock)
		qtyByProductID := make(map[int64]int, len(items))
		for _, item := range items {
			qtyByProductID[item.ProductID] += item.Quantity
		}
		productIDs := make([]int64, 0, len(qtyByProductID))
		for id := range qtyByProductID {
			productIDs = append(productIDs, id)
		}
		slices.Sort(productIDs)

		// FOR UPDATE в отсортированном порядке: lock → validate → reserve
		products := make(map[int64]m.Product, len(productIDs))
		for _, productID := range productIDs {
			product, err := os.productRepo.GetProductByIDForUpdate(ctx, productID)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					return ErrProductNotFound
				}
				return fmt.Errorf("%w: %v", ErrGetProductByID, err)
			}
			if product.DeletedAt != nil {
				return ErrProductDeleted
			}
			qty := qtyByProductID[productID]
			if product.AvailableQuantity() < qty {
				return fmt.Errorf("%w: product %d: available %d, required %d",
					ErrInsufficientStock, productID, product.AvailableQuantity(), qty)
			}
			if err := os.productRepo.ChangeStockAndReserved(ctx, productID, 0, qty); err != nil {
				return fmt.Errorf("%w: %v", ErrUpdateProduct, err)
			}
			products[productID] = product
		}

		itemsBySellerID := make(map[int64][]m.OrderItem)
		for _, item := range items {
			product := products[item.ProductID]
			item.PriceAtPurchase = product.Price
			itemsBySellerID[product.SellerID] = append(itemsBySellerID[product.SellerID], item)
		}

		// single seller optimization
		if len(itemsBySellerID) == 1 {
			var onlySellerID int64
			for id := range itemsBySellerID {
				onlySellerID = id
			}
			sellerItems := itemsBySellerID[onlySellerID]
			for _, item := range sellerItems {
				_, err = os.orderItemRepo.UpdateOrderItem(ctx, item.ID, m.OrderItemUpdate{
					PriceAtPurchase: &item.PriceAtPurchase,
				})
				if err != nil {
					return fmt.Errorf("%w: %v", ErrUpdateOrderItem, err)
				}
			}
			createdOrdersIDs = []int64{cart.ID}

			totalAmount := decimal.Zero
			for _, item := range sellerItems {
				totalAmount = totalAmount.Add(item.PriceAtPurchase.Mul(decimal.NewFromInt(int64(item.Quantity))))
			}
			_, err := os.orderRepo.UpdateOrder(ctx, cart.ID, m.OrderUpdate{
				SellerID:    &onlySellerID,
				AddressID:   &addressID,
				TotalAmount: &totalAmount,
			})
			if err != nil {
				return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
			}
			return nil
		}

		createdOrdersIDs = make([]int64, 0, len(itemsBySellerID))
		for sellerID, sellerItems := range itemsBySellerID {
			totalAmount := decimal.Zero
			for _, item := range sellerItems {
				totalAmount = totalAmount.Add(item.PriceAtPurchase.Mul(decimal.NewFromInt(int64(item.Quantity))))
			}
			orderID, err := os.orderRepo.CreateOrder(ctx, m.OrderCreate{
				UserID:      actor.UserID,
				AddressID:   &addressID,
				SellerID:    &sellerID,
				Status:      checkoutTo,
				TotalAmount: totalAmount,
			})
			if err != nil {
				return fmt.Errorf("%w: %v", ErrCreateOrder, err)
			}
			for _, item := range sellerItems {
				_, err = os.orderItemRepo.CreateOrderItem(ctx, m.OrderItemCreate{
					OrderID:         orderID,
					ProductID:       item.ProductID,
					Quantity:        item.Quantity,
					PriceAtPurchase: item.PriceAtPurchase,
				})
				if err != nil {
					return fmt.Errorf("%w: %v", ErrCreateOrderItem, err)
				}
			}
			createdOrdersIDs = append(createdOrdersIDs, orderID)
		}

		if err := os.orderRepo.DeleteOrderByID(ctx, cart.ID); err != nil {
			return fmt.Errorf("%w: %v", ErrDeleteOrderByID, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return createdOrdersIDs, nil
}

func (os OrderService) PayOrder(ctx context.Context, actor Actor, orderID int64) (err error) {
	if !actor.HasRole(m.RoleBuyer) {
		return ErrPermissionDenied
	}

	order, err := os.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: %v", ErrOrderNotFound, err)
		}
		return fmt.Errorf("%w: %v", ErrGetOrderByID, err)
	}
	if actor.UserID != order.UserID {
		return ErrNotYourOrder
	}
	if err := validateOrderStatusTransition(order.Status, orderTransitionPay); err != nil {
		return err
	}

	from, to := orderTransitionStatuses(orderTransitionPay)
	err = os.orderRepo.UpdateOrderStatus(ctx, orderID, from, to)
	if errors.Is(err, ErrNotFound) {
		return fmt.Errorf("%w: concurrent status change", ErrOrderStatusInvalid)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
	}
	return nil
}

func (os OrderService) CancelOrder(ctx context.Context, actor Actor, orderID int64) (err error) {
	if !actor.IsAdmin() && !actor.HasRole(m.RoleBuyer) {
		return ErrPermissionDenied
	}

	order, err := os.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: %v", ErrOrderNotFound, err)
		}
		return fmt.Errorf("%w: %v", ErrGetOrderByID, err)
	}
	if !actor.IsAdmin() && actor.UserID != order.UserID {
		return ErrNotYourOrder
	}
	if err := validateOrderStatusTransition(order.Status, orderTransitionCancel); err != nil {
		return err
	}

	items, err := os.orderItemRepo.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrGetOrderItemsByOrderID, err)
	}

	// collect unique product IDs for lock ordering
	productIDSet := make(map[int64]struct{}, len(items))
	for _, item := range items {
		productIDSet[item.ProductID] = struct{}{}
	}
	cancelProductIDs := make([]int64, 0, len(productIDSet))
	for id := range productIDSet {
		cancelProductIDs = append(cancelProductIDs, id)
	}
	slices.Sort(cancelProductIDs)

	return os.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		from, to := orderTransitionStatuses(orderTransitionCancel)
		err = os.orderRepo.UpdateOrderStatus(ctx, orderID, from, to)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				// concurrent cancel/expire beat us — order already transitioned
				return fmt.Errorf("%w: concurrent status change", ErrOrderStatusInvalid)
			}
			return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
		}
		// acquire row locks in ascending-ID order before mutating stock; prevents
		// deadlocks when cancel/checkout/ship run concurrently on the same products
		for _, productID := range cancelProductIDs {
			if _, err := os.productRepo.GetProductByIDForUpdate(ctx, productID); err != nil {
				if errors.Is(err, ErrNotFound) {
					return fmt.Errorf("%w: product %d", ErrProductNotFound, productID)
				}
				return fmt.Errorf("%w: product %d: %v", ErrGetProductByID, productID, err)
			}
		}
		for _, item := range items {
			err = os.productRepo.ChangeStockAndReserved(ctx, item.ProductID, 0, -item.Quantity)
			if errors.Is(err, ErrNotFound) {
				return fmt.Errorf("%w: %v", ErrProductNotFound, err)
			}
			if errors.Is(err, ErrStockInvariantViolated) {
				return fmt.Errorf("%w: %v", ErrInsufficientStock, err)
			}
			if err != nil {
				return fmt.Errorf("%w: %v", ErrChangeStockAndReserved, err)
			}
		}
		return nil
	})
}

func (os OrderService) ExpireOrders(ctx context.Context, deadline time.Time) error {
	orders, err := os.orderRepo.GetExpiredPendingOrders(ctx, deadline)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrGetOrdersByUserID, err)
	}
	var expireErrors []error
	for _, order := range orders {
		items, err := os.orderItemRepo.GetOrderItemsByOrderID(ctx, order.ID)
		if err != nil {
			expireErrors = append(expireErrors, &ExpireOrderError{
				OrderID: order.ID,
				Stage:   ExpireStageGetItems,
				Cause:   fmt.Errorf("%w: %v", ErrGetOrderItemsByOrderID, err),
			})
			continue
		}
		// collect unique product IDs in ascending order for deterministic lock ordering
		expireProductIDSet := make(map[int64]struct{}, len(items))
		for _, item := range items {
			expireProductIDSet[item.ProductID] = struct{}{}
		}
		expireProductIDs := make([]int64, 0, len(expireProductIDSet))
		for id := range expireProductIDSet {
			expireProductIDs = append(expireProductIDs, id)
		}
		slices.Sort(expireProductIDs)

		lockedByID := make(map[int64]m.Product, len(expireProductIDs))
		if err = os.txManager.WithTransaction(ctx, func(ctx context.Context) error {
			from, to := orderTransitionStatuses(orderTransitionExpire)
			err = os.orderRepo.UpdateOrderStatus(ctx, order.ID, from, to)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					// order was paid or already cancelled between snapshot and tx — skip
					return nil
				}
				return &ExpireOrderError{
					OrderID: order.ID,
					Stage:   ExpireStageUpdateStatus,
					Cause:   fmt.Errorf("%w: %v", ErrUpdateOrder, err),
				}
			}
			// acquire row locks in ascending-ID order before releasing reserved stock
			for _, productID := range expireProductIDs {
				product, err := os.productRepo.GetProductByIDForUpdate(ctx, productID)
				if err != nil {
					cause := fmt.Errorf("%w: %v", ErrGetProductByID, err)
					if errors.Is(err, ErrNotFound) {
						cause = fmt.Errorf("%w", ErrProductNotFound)
					}
					return &ExpireOrderError{
						OrderID:   order.ID,
						ProductID: productID,
						Stage:     ExpireStageLockProduct,
						Cause:     cause,
					}
				}
				lockedByID[productID] = product
			}
			for _, item := range items {
				err = os.productRepo.ChangeStockAndReserved(ctx, item.ProductID, 0, -item.Quantity)
				if err != nil {
					cause := fmt.Errorf("%w: %v", ErrChangeStockAndReserved, err)
					switch {
					case errors.Is(err, ErrNotFound):
						cause = fmt.Errorf("%w: %v", ErrProductNotFound, err)
					case errors.Is(err, ErrStockInvariantViolated):
						cause = fmt.Errorf("%w: %v", ErrInsufficientStock, err)
					}
					return &ExpireOrderError{
						OrderID:          order.ID,
						ProductID:        item.ProductID,
						Quantity:         item.Quantity,
						ReservedQuantity: lockedByID[item.ProductID].ReservedQuantity,
						Stage:            ExpireStageReleaseStock,
						Cause:            cause,
					}
				}
			}
			return nil
		}); err != nil {
			var ee *ExpireOrderError
			if errors.As(err, &ee) {
				expireErrors = append(expireErrors, err)
			} else {
				expireErrors = append(expireErrors, &ExpireOrderError{
					OrderID: order.ID,
					Stage:   "tx",
					Cause:   err,
				})
			}
			continue
		}
	}
	return errors.Join(expireErrors...)
}

func (os OrderService) ShipOrder(ctx context.Context, actor Actor, orderID int64) (err error) {
	if !actor.HasRole(m.RoleSeller) {
		return ErrPermissionDenied
	}

	seller, err := os.sellerGetter.GetSellerByUserID(ctx, actor.UserID)
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
	if err := validateOrderStatusTransition(order.Status, orderTransitionShip); err != nil {
		return err
	}

	return os.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		from, to := orderTransitionStatuses(orderTransitionShip)
		err = os.orderRepo.UpdateOrderStatus(ctx, orderID, from, to)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return fmt.Errorf("%w: concurrent status change", ErrOrderStatusInvalid)
			}
			return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
		}

		items, err := os.orderItemRepo.GetOrderItemsByOrderID(ctx, orderID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrGetOrderItemsByOrderID, err)
		}
		// collect unique product IDs in ascending order for deterministic lock ordering
		shipProductIDSet := make(map[int64]struct{}, len(items))
		for _, item := range items {
			shipProductIDSet[item.ProductID] = struct{}{}
		}
		shipProductIDs := make([]int64, 0, len(shipProductIDSet))
		for id := range shipProductIDSet {
			shipProductIDs = append(shipProductIDs, id)
		}
		slices.Sort(shipProductIDs)
		// acquire row locks in ascending-ID order before reducing stock
		for _, productID := range shipProductIDs {
			if _, err := os.productRepo.GetProductByIDForUpdate(ctx, productID); err != nil {
				if errors.Is(err, ErrNotFound) {
					return fmt.Errorf("%w: product %d", ErrProductNotFound, productID)
				}
				return fmt.Errorf("%w: product %d: %v", ErrGetProductByID, productID, err)
			}
		}
		for _, item := range items {
			err = os.productRepo.ChangeStockAndReserved(ctx, item.ProductID, -item.Quantity, -item.Quantity)
			if errors.Is(err, ErrNotFound) {
				return fmt.Errorf("%w: %v", ErrProductNotFound, err)
			}
			if errors.Is(err, ErrStockInvariantViolated) {
				return fmt.Errorf("%w: %v", ErrInsufficientStock, err)
			}
			if err != nil {
				return fmt.Errorf("%w: %v", ErrChangeStockAndReserved, err)
			}
		}
		return nil
	})
}

func (os OrderService) DeliverOrder(ctx context.Context, actor Actor, orderID int64) (err error) {
	if !actor.HasRole(m.RoleSeller) {
		return ErrPermissionDenied
	}

	seller, err := os.sellerGetter.GetSellerByUserID(ctx, actor.UserID)
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
	if err := validateOrderStatusTransition(order.Status, orderTransitionDeliver); err != nil {
		return err
	}

	from, to := orderTransitionStatuses(orderTransitionDeliver)
	err = os.orderRepo.UpdateOrderStatus(ctx, orderID, from, to)
	if errors.Is(err, ErrNotFound) {
		return fmt.Errorf("%w: concurrent status change", ErrOrderStatusInvalid)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpdateOrder, err)
	}
	return nil
}
