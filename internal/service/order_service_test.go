package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"
)

var someAddressID int64 = 77
var someSellerID int64 = 10
var someProductID int64 = 200
var someItemID int64 = 300
var someOrderID int64 = 500

var someOrder = m.Order{
	ID:          someOrderID,
	UserID:      someID,
	AddressID:   &someAddressID,
	SellerID:    &someSellerID,
	Status:      m.StatusPending,
	TotalAmount: decimal.NewFromFloat(199.98),
	CreatedAt:   someTime,
	UpdatedAt:   someTime,
}

var someDraftOrder = m.Order{
	ID:          someOrderID,
	UserID:      someID,
	Status:      m.StatusDraft,
	TotalAmount: decimal.Zero,
	CreatedAt:   someTime,
	UpdatedAt:   someTime,
}

var someOrderItem = m.OrderItem{
	ID:              someItemID,
	OrderID:         someOrderID,
	ProductID:       someProductID,
	Quantity:        2,
	PriceAtPurchase: someProductPrice,
}

type MockOrderReturn struct {
	Order m.Order
	Error error
}

type MockOrderListReturn struct {
	Orders []m.Order
	Error  error
}

type MockOrderItemReturn struct {
	Item  m.OrderItem
	Error error
}

type MockOrderItemListReturn struct {
	Items []m.OrderItem
	Error error
}

type passThroughTxManager struct{}

func (passThroughTxManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func newOrderService(ctrl *gomock.Controller) (
	service.OrderService,
	*mock_service.MockOrderRepo,
	*mock_service.MockOrderItemRepo,
	*mock_service.MockProductGetter,
	*mock_service.MockSellerGetter,
	*mock_service.MockProductUpdater,
) {
	orderMock := mock_service.NewMockOrderRepo(ctrl)
	itemMock := mock_service.NewMockOrderItemRepo(ctrl)
	productMock := mock_service.NewMockProductGetter(ctrl)
	productUpdaterMock := mock_service.NewMockProductUpdater(ctrl)
	sellerMock := mock_service.NewMockSellerGetter(ctrl)
	svc := service.NewOrderService(orderMock, itemMock, productMock, productUpdaterMock, sellerMock, passThroughTxManager{})
	return svc, orderMock, itemMock, productMock, sellerMock, productUpdaterMock
}

func assertOrder(t *testing.T, got, want m.Order) {
	t.Helper()
	if got.ID != want.ID || got.UserID != want.UserID || got.Status != want.Status {
		t.Fatalf("invalid order. expected: %+v, got: %+v", want, got)
	}
}

// ==================== GetOrderByID ====================

func TestGetOrderByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, orderMock, _, _, _, _ := newOrderService(ctrl)
	ctx := context.Background()

	type testCase struct {
		Description string
		MockReturn  MockOrderReturn
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			MockReturn:  MockOrderReturn{Order: someOrder},
		},
		{
			Description: "Not found",
			MockReturn:  MockOrderReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrOrderNotFound,
		},
		{
			Description: "Repo error",
			MockReturn:  MockOrderReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetOrderByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(tCase.MockReturn.Order, tCase.MockReturn.Error)

			order, err := svc.GetOrderByID(ctx, someOrderID)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil {
				assertOrder(t, order, someOrder)
			}
		})
	}
}

// ==================== GetOrdersByUserID ====================

func TestGetOrdersByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, orderMock, _, _, _, _ := newOrderService(ctrl)
	ctx := context.Background()

	type testCase struct {
		Description    string
		MockReturn     MockOrderListReturn
		ExpectedErr    error
		ExpectedLength int
	}

	tCases := []testCase{
		{
			Description:    "Success",
			MockReturn:     MockOrderListReturn{Orders: []m.Order{someOrder}},
			ExpectedLength: 1,
		},
		{
			Description:    "Empty list",
			MockReturn:     MockOrderListReturn{Orders: []m.Order{}},
			ExpectedLength: 0,
		},
		{
			Description: "Repo error",
			MockReturn:  MockOrderListReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetOrdersByUserID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			orderMock.EXPECT().GetOrdersByUserID(ctx, someID, m.PaginationOpts{}).Return(tCase.MockReturn.Orders, tCase.MockReturn.Error)

			orders, err := svc.GetOrdersByUserID(ctx, someID, m.PaginationOpts{})
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && len(orders) != tCase.ExpectedLength {
				t.Fatalf("expected %d orders, got %d", tCase.ExpectedLength, len(orders))
			}
		})
	}
}

// ==================== GetOrderItemsByOrderID ====================

func TestGetOrderItemsByOrderID(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, itemMock, _, _, _ := newOrderService(ctrl)
	ctx := context.Background()

	type testCase struct {
		Description    string
		MockReturn     MockOrderItemListReturn
		ExpectedErr    error
		ExpectedLength int
	}

	tCases := []testCase{
		{
			Description:    "Success",
			MockReturn:     MockOrderItemListReturn{Items: []m.OrderItem{someOrderItem}},
			ExpectedLength: 1,
		},
		{
			Description:    "Empty list",
			MockReturn:     MockOrderItemListReturn{Items: []m.OrderItem{}},
			ExpectedLength: 0,
		},
		{
			Description: "Repo error",
			MockReturn:  MockOrderItemListReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetOrderItemsByOrderID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			itemMock.EXPECT().GetOrderItemsByOrderID(ctx, someOrderID).Return(tCase.MockReturn.Items, tCase.MockReturn.Error)

			items, err := svc.GetOrderItemsByOrderID(ctx, someOrderID)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && len(items) != tCase.ExpectedLength {
				t.Fatalf("expected %d items, got %d", tCase.ExpectedLength, len(items))
			}
		})
	}
}

// ==================== GetSellerOrdersByUserID ====================

func TestGetSellerOrdersByUserID(t *testing.T) {
	ctx := context.Background()

	seller := m.Seller{ID: someSellerID, UserID: someID}
	sellerOrders := []m.Order{someOrder}
	pg := m.PaginationOpts{Page: 1, Limit: 10}

	type testCase struct {
		Description    string
		MockSeller     MockSellerReturn
		MockOrders     *MockOrderListReturn
		ExpectedErr    error
		ExpectedLength int
	}

	tCases := []testCase{
		{
			Description:    "Success",
			MockSeller:     MockSellerReturn{Seller: seller},
			MockOrders:     &MockOrderListReturn{Orders: sellerOrders},
			ExpectedLength: 1,
		},
		{
			Description:    "Success empty orders",
			MockSeller:     MockSellerReturn{Seller: seller},
			MockOrders:     &MockOrderListReturn{Orders: []m.Order{}},
			ExpectedLength: 0,
		},
		{
			Description: "Seller not found",
			MockSeller:  MockSellerReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrSellerNotFound,
		},
		{
			Description: "GetSellerByUserID repo error",
			MockSeller:  MockSellerReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetSellerByUserID,
		},
		{
			Description: "GetSellerOrdersBySellerID repo error",
			MockSeller:  MockSellerReturn{Seller: seller},
			MockOrders:  &MockOrderListReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetOrdersBySellerID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc, orderMock, _, _, sellerMock, _ := newOrderService(ctrl)

			sellerMock.EXPECT().GetSellerByUserID(ctx, someID).Return(tCase.MockSeller.Seller, tCase.MockSeller.Error)

			if tCase.MockOrders != nil {
				orderMock.EXPECT().GetSellerOrdersBySellerID(ctx, seller.ID, pg).Return(tCase.MockOrders.Orders, tCase.MockOrders.Error)
			}

			orders, err := svc.GetSellerOrdersByUserID(ctx, someID, pg)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && len(orders) != tCase.ExpectedLength {
				t.Fatalf("expected %d orders, got %d", tCase.ExpectedLength, len(orders))
			}
		})
	}
}

// ==================== GetCart ====================

func TestGetCart(t *testing.T) {
	ctx := context.Background()

	olderDraft := m.Order{
		ID:        someOrderID + 1,
		UserID:    someID,
		Status:    m.StatusDraft,
		CreatedAt: someTime.Add(-time.Hour),
	}
	newerDraft := m.Order{
		ID:        someOrderID,
		UserID:    someID,
		Status:    m.StatusDraft,
		CreatedAt: someTime,
	}
	paidOrder := m.Order{
		ID:     someOrderID + 2,
		UserID: someID,
		Status: m.StatusPaid,
	}

	cartItems := []m.OrderItem{someOrderItem}
	expectedTotal := someOrderItem.PriceAtPurchase.Mul(decimal.NewFromInt(int64(someOrderItem.Quantity)))

	type testCase struct {
		Description     string
		MockReturn      MockOrderListReturn
		MockItems       *MockOrderItemListReturn
		ExpectedErr     error
		ExpectedOrderID *int64
		ExpectedTotal   *decimal.Decimal
	}

	tCases := []testCase{
		{
			Description:     "Single draft",
			MockReturn:      MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockItems:       &MockOrderItemListReturn{Items: cartItems},
			ExpectedOrderID: &someDraftOrder.ID,
			ExpectedTotal:   &expectedTotal,
		},
		{
			Description:     "Multiple drafts returns latest",
			MockReturn:      MockOrderListReturn{Orders: []m.Order{olderDraft, newerDraft}},
			MockItems:       &MockOrderItemListReturn{Items: []m.OrderItem{}},
			ExpectedOrderID: &newerDraft.ID,
		},
		{
			Description:     "Mixed statuses picks draft",
			MockReturn:      MockOrderListReturn{Orders: []m.Order{paidOrder, newerDraft}},
			MockItems:       &MockOrderItemListReturn{Items: []m.OrderItem{}},
			ExpectedOrderID: &newerDraft.ID,
		},
		{
			Description:     "Empty cart computes zero total",
			MockReturn:      MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockItems:       &MockOrderItemListReturn{Items: []m.OrderItem{}},
			ExpectedOrderID: &someDraftOrder.ID,
		},
		{
			Description: "GetItems repo error inside GetCart",
			MockReturn:  MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockItems:   &MockOrderItemListReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetCart,
		},
		{
			Description: "No orders at all",
			MockReturn:  MockOrderListReturn{Orders: []m.Order{}},
			ExpectedErr: service.ErrCartNotFound,
		},
		{
			Description: "No draft orders",
			MockReturn:  MockOrderListReturn{Orders: []m.Order{paidOrder}},
			ExpectedErr: service.ErrCartNotFound,
		},
		{
			Description: "Repo error",
			MockReturn:  MockOrderListReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetCart,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc, orderMock, itemMock, _, _, _ := newOrderService(ctrl)

			orderMock.EXPECT().GetOrdersByUserID(ctx, someID, m.PaginationOpts{}).Return(tCase.MockReturn.Orders, tCase.MockReturn.Error)

			if tCase.MockItems != nil {
				itemMock.EXPECT().GetOrderItemsByOrderID(ctx, gomock.Any()).Return(tCase.MockItems.Items, tCase.MockItems.Error)
			}

			cart, err := svc.GetCart(ctx, someID)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedOrderID != nil {
				if cart.ID != *tCase.ExpectedOrderID {
					t.Fatalf("expected cart ID %d, got %d", *tCase.ExpectedOrderID, cart.ID)
				}
			}
			if tCase.ExpectedTotal != nil {
				if !cart.TotalAmount.Equal(*tCase.ExpectedTotal) {
					t.Fatalf("expected total %s, got %s", tCase.ExpectedTotal.String(), cart.TotalAmount.String())
				}
			}
		})
	}
}

// ==================== AddItemToCart ====================

func TestAddItemToCart(t *testing.T) {
	ctx := context.Background()

	quantity := 2
	productWithStock := m.Product{
		ID:            someProductID,
		SellerID:      someSellerID,
		Price:         someProductPrice,
		StockQuantity: 50,
	}

	emptyItems := MockOrderItemListReturn{Items: []m.OrderItem{}}
	existingItemSameProduct := MockOrderItemListReturn{Items: []m.OrderItem{someOrderItem}}
	otherProductItem := m.OrderItem{ID: someItemID + 1, OrderID: someOrderID, ProductID: someProductID + 1, Quantity: 1, PriceAtPurchase: someProductPrice}
	existingItemDiffProduct := MockOrderItemListReturn{Items: []m.OrderItem{otherProductItem}}

	type testCase struct {
		Description      string
		MockGetOrders    MockOrderListReturn
		MockGetCartItems *MockOrderItemListReturn // items returned inside GetCart (for total)
		MockCreateOrder  *MockCreateReturn
		MockGetItems     *MockOrderItemListReturn // items returned for duplicate check
		MockGetProduct   *MockProductReturn
		MockCreateItem   *MockCreateReturn
		ExpectedErr      error
	}

	tCases := []testCase{
		{
			Description:      "Success with existing cart, no items",
			MockGetOrders:    MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetCartItems: &emptyItems,
			MockGetItems:     &emptyItems,
			MockGetProduct:   &MockProductReturn{Product: productWithStock},
			MockCreateItem:   &MockCreateReturn{ID: someItemID},
		},
		{
			Description:      "Success with existing cart, different product in cart",
			MockGetOrders:    MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetCartItems: &existingItemDiffProduct,
			MockGetItems:     &existingItemDiffProduct,
			MockGetProduct:   &MockProductReturn{Product: productWithStock},
			MockCreateItem:   &MockCreateReturn{ID: someItemID},
		},
		{
			Description:     "Success creating new cart",
			MockGetOrders:   MockOrderListReturn{Orders: []m.Order{}},
			MockCreateOrder: &MockCreateReturn{ID: someOrderID},
			MockGetProduct:  &MockProductReturn{Product: productWithStock},
			MockCreateItem:  &MockCreateReturn{ID: someItemID},
		},
		{
			Description:   "GetOrders repo error",
			MockGetOrders: MockOrderListReturn{Error: errors.New("db error")},
			ExpectedErr:   service.ErrGetCart,
		},
		{
			Description:     "CreateOrder fails when no cart",
			MockGetOrders:   MockOrderListReturn{Orders: []m.Order{}},
			MockGetProduct:  &MockProductReturn{Product: productWithStock},
			MockCreateOrder: &MockCreateReturn{Error: errors.New("db error")},
			ExpectedErr:     service.ErrCreateOrder,
		},
		{
			Description:      "Product already in cart",
			MockGetOrders:    MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetCartItems: &existingItemSameProduct,
			MockGetItems:     &existingItemSameProduct,
			ExpectedErr:      service.ErrProductAlreadyInCart,
		},
		{
			Description:      "GetItems repo error for duplicate check",
			MockGetOrders:    MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetCartItems: &emptyItems,
			MockGetItems:     &MockOrderItemListReturn{Error: errors.New("db error")},
			ExpectedErr:      service.ErrGetOrderItemsByOrderID,
		},
		{
			Description:      "Product not found",
			MockGetOrders:    MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetCartItems: &emptyItems,
			MockGetItems:     &emptyItems,
			MockGetProduct:   &MockProductReturn{Error: service.ErrNotFound},
			ExpectedErr:      service.ErrProductNotFound,
		},
		{
			Description:      "GetProduct repo error",
			MockGetOrders:    MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetCartItems: &emptyItems,
			MockGetItems:     &emptyItems,
			MockGetProduct:   &MockProductReturn{Error: errors.New("db error")},
			ExpectedErr:      service.ErrGetProductByID,
		},
		{
			Description:      "Quantity exceeds stock",
			MockGetOrders:    MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetCartItems: &emptyItems,
			MockGetItems:     &emptyItems,
			MockGetProduct: &MockProductReturn{Product: m.Product{
				ID: someProductID, StockQuantity: 1, Price: someProductPrice,
			}},
			ExpectedErr: service.ErrQuantityTooBig,
		},
		{
			Description:      "CreateOrderItem fails",
			MockGetOrders:    MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetCartItems: &emptyItems,
			MockGetItems:     &emptyItems,
			MockGetProduct:   &MockProductReturn{Product: productWithStock},
			MockCreateItem:   &MockCreateReturn{Error: errors.New("db error")},
			ExpectedErr:      service.ErrCreateOrderItem,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc, orderMock, itemMock, productMock, _, _ := newOrderService(ctrl)

			orderMock.EXPECT().GetOrdersByUserID(ctx, someID, m.PaginationOpts{}).Return(tCase.MockGetOrders.Orders, tCase.MockGetOrders.Error)

			if tCase.MockCreateOrder != nil {
				orderMock.EXPECT().CreateOrder(ctx, gomock.Any()).Return(tCase.MockCreateOrder.ID, tCase.MockCreateOrder.Error)
			}
			// GetCart internally calls GetOrderItemsByOrderID for total computation
			if tCase.MockGetCartItems != nil {
				itemMock.EXPECT().GetOrderItemsByOrderID(ctx, someOrderID).Return(tCase.MockGetCartItems.Items, tCase.MockGetCartItems.Error)
			}
			// AddItemToCart calls GetOrderItemsByOrderID for duplicate check
			if tCase.MockGetItems != nil {
				itemMock.EXPECT().GetOrderItemsByOrderID(ctx, someOrderID).Return(tCase.MockGetItems.Items, tCase.MockGetItems.Error)
			}
			if tCase.MockGetProduct != nil {
				productMock.EXPECT().GetProductByID(ctx, someProductID).Return(tCase.MockGetProduct.Product, tCase.MockGetProduct.Error)
			}
			if tCase.MockCreateItem != nil {
				itemMock.EXPECT().CreateOrderItem(ctx, gomock.Any()).Return(tCase.MockCreateItem.ID, tCase.MockCreateItem.Error)
			}

			err := svc.AddItemToCart(ctx, someID, someProductID, quantity)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

// ==================== ChangeQuantityCartItem ====================

func TestChangeQuantityCartItem(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, itemMock, productMock, _, _ := newOrderService(ctrl)
	ctx := context.Background()

	newQuantity := 3
	productWithStock := m.Product{
		ID:            someProductID,
		StockQuantity: 50,
		Price:         someProductPrice,
	}

	type testCase struct {
		Description    string
		MockGetItem    MockOrderItemReturn
		MockGetProduct *MockProductReturn
		MockUpdate     *MockOrderItemReturn
		ExpectedErr    error
	}

	tCases := []testCase{
		{
			Description:    "Success",
			MockGetItem:    MockOrderItemReturn{Item: someOrderItem},
			MockGetProduct: &MockProductReturn{Product: productWithStock},
			MockUpdate:     &MockOrderItemReturn{Item: someOrderItem},
		},
		{
			Description: "Item not found",
			MockGetItem: MockOrderItemReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrOrderItemNotFound,
		},
		{
			Description: "GetItem repo error",
			MockGetItem: MockOrderItemReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetOrderItemByID,
		},
		{
			Description:    "Product not found",
			MockGetItem:    MockOrderItemReturn{Item: someOrderItem},
			MockGetProduct: &MockProductReturn{Error: service.ErrNotFound},
			ExpectedErr:    service.ErrProductNotFound,
		},
		{
			Description:    "GetProduct repo error",
			MockGetItem:    MockOrderItemReturn{Item: someOrderItem},
			MockGetProduct: &MockProductReturn{Error: errors.New("db error")},
			ExpectedErr:    service.ErrGetProductByID,
		},
		{
			Description: "Quantity exceeds stock",
			MockGetItem: MockOrderItemReturn{Item: someOrderItem},
			MockGetProduct: &MockProductReturn{Product: m.Product{
				ID: someProductID, StockQuantity: 1, Price: someProductPrice,
			}},
			ExpectedErr: service.ErrQuantityTooBig,
		},
		{
			Description:    "UpdateOrderItem fails",
			MockGetItem:    MockOrderItemReturn{Item: someOrderItem},
			MockGetProduct: &MockProductReturn{Product: productWithStock},
			MockUpdate:     &MockOrderItemReturn{Error: errors.New("db error")},
			ExpectedErr:    service.ErrUpdateOrderItem,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			itemMock.EXPECT().GetOrderItemByID(ctx, someItemID).Return(tCase.MockGetItem.Item, tCase.MockGetItem.Error)

			if tCase.MockGetProduct != nil {
				productMock.EXPECT().GetProductByID(ctx, someOrderItem.ProductID).Return(tCase.MockGetProduct.Product, tCase.MockGetProduct.Error)
			}
			if tCase.MockUpdate != nil {
				itemMock.EXPECT().UpdateOrderItem(ctx, someItemID, gomock.Any()).Return(tCase.MockUpdate.Item, tCase.MockUpdate.Error)
			}

			err := svc.ChangeQuantityCartItem(ctx, someItemID, newQuantity)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

// ==================== DeleteCartItem ====================

func TestDeleteCartItem(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, itemMock, _, _, _ := newOrderService(ctrl)
	ctx := context.Background()

	type testCase struct {
		Description string
		MockError   error
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
		},
		{
			Description: "Repo error",
			MockError:   errors.New("db error"),
			ExpectedErr: service.ErrDeleteOrderItemByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			itemMock.EXPECT().DeleteOrderItemByID(ctx, someItemID).Return(tCase.MockError)

			err := svc.DeleteCartItem(ctx, someItemID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

// ==================== Checkout ====================

func TestCheckout(t *testing.T) {
	ctx := context.Background()

	product1 := m.Product{ID: 200, SellerID: 10, Price: decimal.NewFromFloat(50.00)}
	product2 := m.Product{ID: 201, SellerID: 20, Price: decimal.NewFromFloat(30.00)}

	item1 := m.OrderItem{ID: 300, OrderID: someOrderID, ProductID: 200, Quantity: 2, PriceAtPurchase: product1.Price}
	item2 := m.OrderItem{ID: 301, OrderID: someOrderID, ProductID: 201, Quantity: 1, PriceAtPurchase: product2.Price}

	type testCase struct {
		Description    string
		MockGetOrders  MockOrderListReturn
		MockGetItems   *MockOrderItemListReturn
		MockProducts   map[int64]MockProductReturn
		MockCreateOrds []MockCreateReturn
		MockCreateItem *error
		MockDeleteCart *error
		ExpectedErr    error
		ExpectedCount  int
	}

	tCases := []testCase{
		{
			Description:   "Success single seller",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1}},
			MockProducts: map[int64]MockProductReturn{
				200: {Product: product1},
			},
			MockCreateOrds: []MockCreateReturn{{ID: 600}},
			MockCreateItem: ptrErr(nil),
			MockDeleteCart: ptrErr(nil),
			ExpectedCount:  1,
		},
		{
			Description:   "Success two sellers splits orders",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1, item2}},
			MockProducts: map[int64]MockProductReturn{
				200: {Product: product1},
				201: {Product: product2},
			},
			MockCreateOrds: []MockCreateReturn{{ID: 600}, {ID: 601}},
			MockCreateItem: ptrErr(nil),
			MockDeleteCart: ptrErr(nil),
			ExpectedCount:  2,
		},
		{
			Description:   "No cart",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{}},
			ExpectedErr:   service.ErrCartNotFound,
		},
		{
			Description:   "GetOrders repo error",
			MockGetOrders: MockOrderListReturn{Error: errors.New("db error")},
			ExpectedErr:   service.ErrGetCart,
		},
		{
			Description:   "Empty cart",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{}},
			ExpectedErr:   service.ErrEmptyCart,
		},
		{
			Description:   "GetItems repo error",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetItems:  &MockOrderItemListReturn{Error: errors.New("db error")},
			ExpectedErr:   service.ErrGetOrderItemsByOrderID,
		},
		{
			Description:   "Product not found during checkout",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1}},
			MockProducts: map[int64]MockProductReturn{
				200: {Error: service.ErrNotFound},
			},
			ExpectedErr: service.ErrProductNotFound,
		},
		{
			Description:   "CreateOrder fails",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1}},
			MockProducts: map[int64]MockProductReturn{
				200: {Product: product1},
			},
			MockCreateOrds: []MockCreateReturn{{Error: errors.New("db error")}},
			ExpectedErr:    service.ErrCreateOrder,
		},
		{
			Description:   "CreateOrderItem fails during checkout",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1}},
			MockProducts: map[int64]MockProductReturn{
				200: {Product: product1},
			},
			MockCreateOrds: []MockCreateReturn{{ID: 600}},
			MockCreateItem: ptrErr(errors.New("db error")),
			ExpectedErr:    service.ErrCreateOrderItem,
		},
		{
			Description:   "DeleteCart fails after orders created",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1}},
			MockProducts: map[int64]MockProductReturn{
				200: {Product: product1},
			},
			MockCreateOrds: []MockCreateReturn{{ID: 600}},
			MockCreateItem: ptrErr(nil),
			MockDeleteCart: ptrErr(errors.New("db error")),
			ExpectedErr:    service.ErrDeleteOrderByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc, orderMock, itemMock, productMock, _, _ := newOrderService(ctrl)

			orderMock.EXPECT().GetOrdersByUserID(ctx, someID, m.PaginationOpts{}).Return(tCase.MockGetOrders.Orders, tCase.MockGetOrders.Error)

			// GetCart internally calls GetOrderItemsByOrderID for total computation
			if tCase.MockGetOrders.Error == nil && len(tCase.MockGetOrders.Orders) > 0 {
				hasDraft := false
				for _, o := range tCase.MockGetOrders.Orders {
					if o.Status == m.StatusDraft {
						hasDraft = true
						break
					}
				}
				if hasDraft {
					itemMock.EXPECT().GetOrderItemsByOrderID(ctx, someDraftOrder.ID).Return([]m.OrderItem{}, nil)
				}
			}

			if tCase.MockGetItems != nil {
				itemMock.EXPECT().GetOrderItemsByOrderID(ctx, someDraftOrder.ID).Return(tCase.MockGetItems.Items, tCase.MockGetItems.Error)
			}
			for pid, ret := range tCase.MockProducts {
				productMock.EXPECT().GetProductByID(ctx, pid).Return(ret.Product, ret.Error)
			}
			for _, cr := range tCase.MockCreateOrds {
				orderMock.EXPECT().CreateOrder(ctx, gomock.Any()).Return(cr.ID, cr.Error)
			}
			if tCase.MockCreateItem != nil {
				itemMock.EXPECT().CreateOrderItem(ctx, gomock.Any()).Return(int64(0), *tCase.MockCreateItem).AnyTimes()
			}
			if tCase.MockDeleteCart != nil {
				orderMock.EXPECT().DeleteOrderByID(ctx, someDraftOrder.ID).Return(*tCase.MockDeleteCart)
			}

			orderIDs, err := svc.Checkout(ctx, someID, someAddressID)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && len(orderIDs) != tCase.ExpectedCount {
				t.Fatalf("expected %d order IDs, got %d", tCase.ExpectedCount, len(orderIDs))
			}
		})
	}
}

// ==================== PayOrder ====================

func TestPayOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, orderMock, _, _, _, _ := newOrderService(ctrl)
	ctx := context.Background()

	pendingOrder := someOrder // StatusPending, UserID = someID
	paidOrder := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusPaid}
	otherUsersOrder := m.Order{ID: someOrderID, UserID: otherUserID, Status: m.StatusPending}

	type testCase struct {
		Description string
		MockGet     MockOrderReturn
		MockUpdate  *MockOrderReturn
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			MockGet:     MockOrderReturn{Order: pendingOrder},
			MockUpdate:  &MockOrderReturn{Order: paidOrder},
		},
		{
			Description: "Order not found",
			MockGet:     MockOrderReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrOrderNotFound,
		},
		{
			Description: "GetOrder repo error",
			MockGet:     MockOrderReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetOrderByID,
		},
		{
			Description: "Not your order",
			MockGet:     MockOrderReturn{Order: otherUsersOrder},
			ExpectedErr: service.ErrNotYourOrder,
		},
		{
			Description: "Wrong status - already paid",
			MockGet:     MockOrderReturn{Order: paidOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "Wrong status - shipped",
			MockGet:     MockOrderReturn{Order: m.Order{ID: someOrderID, UserID: someID, Status: m.StatusShipped}},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "UpdateOrder fails",
			MockGet:     MockOrderReturn{Order: pendingOrder},
			MockUpdate:  &MockOrderReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrUpdateOrder,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(tCase.MockGet.Order, tCase.MockGet.Error)

			if tCase.MockUpdate != nil {
				orderMock.EXPECT().UpdateOrder(ctx, someOrderID, gomock.Any()).Return(tCase.MockUpdate.Order, tCase.MockUpdate.Error)
			}

			err := svc.PayOrder(ctx, someOrderID, someID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

// ==================== CancelOrder ====================

func TestCancelOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, orderMock, _, _, _, _ := newOrderService(ctrl)
	ctx := context.Background()

	pendingOrder := someOrder
	paidOrder := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusPaid}
	shippedOrder := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusShipped}
	deliveredOrder := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusDelivered}
	cancelledOrder := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusCancelled}
	otherUsersOrder := m.Order{ID: someOrderID, UserID: otherUserID, Status: m.StatusPending}
	cancelledResult := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusCancelled}

	type testCase struct {
		Description string
		MockGet     MockOrderReturn
		MockUpdate  *MockOrderReturn
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success cancel pending",
			MockGet:     MockOrderReturn{Order: pendingOrder},
			MockUpdate:  &MockOrderReturn{Order: cancelledResult},
		},
		{
			Description: "Success cancel paid",
			MockGet:     MockOrderReturn{Order: paidOrder},
			MockUpdate:  &MockOrderReturn{Order: cancelledResult},
		},
		{
			Description: "Order not found",
			MockGet:     MockOrderReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrOrderNotFound,
		},
		{
			Description: "GetOrder repo error",
			MockGet:     MockOrderReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetOrderByID,
		},
		{
			Description: "Not your order",
			MockGet:     MockOrderReturn{Order: otherUsersOrder},
			ExpectedErr: service.ErrNotYourOrder,
		},
		{
			Description: "Cannot cancel shipped",
			MockGet:     MockOrderReturn{Order: shippedOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "Cannot cancel delivered",
			MockGet:     MockOrderReturn{Order: deliveredOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "Cannot cancel already cancelled",
			MockGet:     MockOrderReturn{Order: cancelledOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "UpdateOrder fails",
			MockGet:     MockOrderReturn{Order: pendingOrder},
			MockUpdate:  &MockOrderReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrUpdateOrder,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(tCase.MockGet.Order, tCase.MockGet.Error)

			if tCase.MockUpdate != nil {
				orderMock.EXPECT().UpdateOrder(ctx, someOrderID, gomock.Any()).Return(tCase.MockUpdate.Order, tCase.MockUpdate.Error)
			}

			err := svc.CancelOrder(ctx, someOrderID, someID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

// ==================== ShipOrder ====================

func TestShipOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, orderMock, itemMock, productMock, sellerMock, productUpdaterMock := newOrderService(ctrl)
	ctx := context.Background()

	seller := m.Seller{ID: someSellerID, UserID: someID}
	paidOrderWithSeller := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusPaid}
	pendingOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusPending}
	alreadyShippedOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusShipped}
	deliveredOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusDelivered}
	otherSellerID := int64(999)
	otherSellersOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &otherSellerID, Status: m.StatusPaid}
	orderNoSeller := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: nil, Status: m.StatusPaid}
	shippedResult := m.Order{ID: someOrderID, Status: m.StatusShipped}

	someOrderItems := []m.OrderItem{
		{ID: 1, OrderID: someOrderID, ProductID: someProductID, Quantity: 2, PriceAtPurchase: someProductPrice},
	}
	someProduct := m.Product{ID: someProductID, StockQuantity: 10, Price: someProductPrice}
	updatedProduct := someProduct
	newQty := someProduct.StockQuantity - someOrderItems[0].Quantity
	updatedProduct.StockQuantity = newQty

	type testCase struct {
		Description    string
		MockSeller     *MockSellerReturn
		MockGet        *MockOrderReturn
		MockUpdate     *MockOrderReturn
		MockItems      bool
		MockProduct    bool
		MockProductUpd bool
		ExpectedErr    error
	}

	tCases := []testCase{
		{
			Description:    "Success",
			MockSeller:     &MockSellerReturn{Seller: seller},
			MockGet:        &MockOrderReturn{Order: paidOrderWithSeller},
			MockUpdate:     &MockOrderReturn{Order: shippedResult},
			MockItems:      true,
			MockProduct:    true,
			MockProductUpd: true,
		},
		{
			Description: "Seller not found",
			MockSeller:  &MockSellerReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrSellerNotFound,
		},
		{
			Description: "GetSeller repo error",
			MockSeller:  &MockSellerReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetSellerByUserID,
		},
		{
			Description: "Order not found",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrOrderNotFound,
		},
		{
			Description: "GetOrder repo error",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetOrderByID,
		},
		{
			Description: "Not your order - different seller",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: otherSellersOrder},
			ExpectedErr: service.ErrNotYourOrder,
		},
		{
			Description: "Not your order - no seller on order",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: orderNoSeller},
			ExpectedErr: service.ErrNotYourOrder,
		},
		{
			Description: "Wrong status - pending",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: pendingOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "Wrong status - already shipped",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: alreadyShippedOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "Wrong status - delivered",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: deliveredOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "UpdateOrder fails",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: paidOrderWithSeller},
			MockUpdate:  &MockOrderReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrUpdateOrder,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			if tCase.MockSeller != nil {
				sellerMock.EXPECT().GetSellerByUserID(ctx, someID).Return(tCase.MockSeller.Seller, tCase.MockSeller.Error)
			}
			if tCase.MockGet != nil {
				orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(tCase.MockGet.Order, tCase.MockGet.Error)
			}
			if tCase.MockUpdate != nil {
				orderMock.EXPECT().UpdateOrder(ctx, someOrderID, gomock.Any()).Return(tCase.MockUpdate.Order, tCase.MockUpdate.Error)
			}
			if tCase.MockItems {
				itemMock.EXPECT().GetOrderItemsByOrderID(ctx, someOrderID).Return(someOrderItems, nil)
			}
			if tCase.MockProduct {
				productMock.EXPECT().GetProductByID(ctx, someProductID).Return(someProduct, nil)
			}
			if tCase.MockProductUpd {
				productUpdaterMock.EXPECT().UpdateProduct(ctx, someProductID, m.ProductUpdate{StockQuantity: &newQty}).Return(updatedProduct, nil)
			}

			err := svc.ShipOrder(ctx, someOrderID, someID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

// ==================== DeliverOrder ====================

func TestDeliverOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, orderMock, _, _, sellerMock, _ := newOrderService(ctrl)
	ctx := context.Background()

	seller := m.Seller{ID: someSellerID, UserID: someID}
	shippedOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusShipped}
	paidOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusPaid}
	pendingOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusPending}
	alreadyDeliveredOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusDelivered}
	otherSellerID := int64(999)
	otherSellersOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &otherSellerID, Status: m.StatusShipped}
	orderNoSeller := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: nil, Status: m.StatusShipped}
	deliveredResult := m.Order{ID: someOrderID, Status: m.StatusDelivered}

	type testCase struct {
		Description string
		MockSeller  *MockSellerReturn
		MockGet     *MockOrderReturn
		MockUpdate  *MockOrderReturn
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: shippedOrder},
			MockUpdate:  &MockOrderReturn{Order: deliveredResult},
		},
		{
			Description: "Seller not found",
			MockSeller:  &MockSellerReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrSellerNotFound,
		},
		{
			Description: "GetSeller repo error",
			MockSeller:  &MockSellerReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetSellerByUserID,
		},
		{
			Description: "Order not found",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrOrderNotFound,
		},
		{
			Description: "GetOrder repo error",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetOrderByID,
		},
		{
			Description: "Not your order - different seller",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: otherSellersOrder},
			ExpectedErr: service.ErrNotYourOrder,
		},
		{
			Description: "Not your order - no seller on order",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: orderNoSeller},
			ExpectedErr: service.ErrNotYourOrder,
		},
		{
			Description: "Wrong status - paid not shipped",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: paidOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "Wrong status - pending",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: pendingOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "Wrong status - already delivered",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: alreadyDeliveredOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description: "UpdateOrder fails",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: shippedOrder},
			MockUpdate:  &MockOrderReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrUpdateOrder,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			if tCase.MockSeller != nil {
				sellerMock.EXPECT().GetSellerByUserID(ctx, someID).Return(tCase.MockSeller.Seller, tCase.MockSeller.Error)
			}
			if tCase.MockGet != nil {
				orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(tCase.MockGet.Order, tCase.MockGet.Error)
			}
			if tCase.MockUpdate != nil {
				orderMock.EXPECT().UpdateOrder(ctx, someOrderID, gomock.Any()).Return(tCase.MockUpdate.Order, tCase.MockUpdate.Error)
			}

			err := svc.DeliverOrder(ctx, someOrderID, someID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}
