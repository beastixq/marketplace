package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/beastixq/marketplace/internal/testsupport"
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

func newOrderService(ctrl *gomock.Controller) (
	service.OrderService,
	*mock_service.MockOrderRepo,
	*mock_service.MockOrderItemRepo,
	*mock_service.MockProductRepo,
	*mock_service.MockSellerGetter,
	*mock_service.MockAddressRepo,
) {
	orderMock := mock_service.NewMockOrderRepo(ctrl)
	itemMock := mock_service.NewMockOrderItemRepo(ctrl)
	productMock := mock_service.NewMockProductRepo(ctrl)
	addressMock := mock_service.NewMockAddressRepo(ctrl)
	sellerMock := mock_service.NewMockSellerGetter(ctrl)
	svc := service.NewOrderService(orderMock, itemMock, productMock, addressMock, sellerMock, testsupport.PassThroughTxManager{})
	return svc, orderMock, itemMock, productMock, sellerMock, addressMock
}

func assertOrder(t *testing.T, got, want m.Order) {
	t.Helper()
	if got.ID != want.ID || got.UserID != want.UserID || got.Status != want.Status {
		t.Fatalf("invalid order. expected: %+v, got: %+v", want, got)
	}
}

func TestOrderService_GetOrderByID(t *testing.T) {
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
			ctrl := gomock.NewController(t)
			svc, orderMock, _, _, _, _ := newOrderService(ctrl)

			orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(tCase.MockReturn.Order, tCase.MockReturn.Error)

			order, err := svc.GetOrderByID(ctx, testActor(someID, m.RoleBuyer), someOrderID)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil {
				assertOrder(t, order, someOrder)
			}
		})
	}
}

func TestOrderService_GetOrdersByUserID(t *testing.T) {
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
			ctrl := gomock.NewController(t)
			svc, orderMock, _, _, _, _ := newOrderService(ctrl)

			orderMock.EXPECT().GetOrdersByUserID(ctx, someID, m.PaginationOpts{}).Return(tCase.MockReturn.Orders, tCase.MockReturn.Error)

			orders, err := svc.GetOrdersByUserID(ctx, testActor(someID, m.RoleBuyer), m.PaginationOpts{})
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && len(orders) != tCase.ExpectedLength {
				t.Fatalf("expected %d orders, got %d", tCase.ExpectedLength, len(orders))
			}
		})
	}
}

func TestOrderService_GetOrderItemsByOrderID(t *testing.T) {
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
			ctrl := gomock.NewController(t)
			svc, orderMock, itemMock, _, _, _ := newOrderService(ctrl)

			orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(someOrder, nil)
			itemMock.EXPECT().GetOrderItemsByOrderID(ctx, someOrderID).Return(tCase.MockReturn.Items, tCase.MockReturn.Error)

			items, err := svc.GetOrderItemsByOrderID(ctx, testActor(someID, m.RoleBuyer), someOrderID)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && len(items) != tCase.ExpectedLength {
				t.Fatalf("expected %d items, got %d", tCase.ExpectedLength, len(items))
			}
		})
	}
}

func TestOrderService_GetSellerOrdersByUserID(t *testing.T) {
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

			orders, err := svc.GetSellerOrdersByUserID(ctx, testActor(someID, m.RoleSeller), pg)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && len(orders) != tCase.ExpectedLength {
				t.Fatalf("expected %d orders, got %d", tCase.ExpectedLength, len(orders))
			}
		})
	}
}

func TestOrderService_GetCart(t *testing.T) {
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

			cart, err := svc.GetCart(ctx, testActor(someID, m.RoleBuyer))
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

func TestOrderService_AddItemToCart(t *testing.T) {
	t.Run("seller is forbidden", func(t *testing.T) {
		svc, _, _, _, _, _ := newOrderService(gomock.NewController(t))

		err := svc.AddItemToCart(context.Background(), testActor(someID, m.RoleSeller), someProductID, 1)

		assertError(t, err, service.ErrPermissionDenied)
	})
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

			orderMock.EXPECT().LockUserCart(ctx, someID).Return(nil)
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

			err := svc.AddItemToCart(ctx, testActor(someID, m.RoleBuyer), someProductID, quantity)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

func TestOrderService_ChangeQuantityCartItem(t *testing.T) {
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
		MockLockErr    *error
		MockUpdateErr  *error
		ExpectedErr    error
	}

	tCases := []testCase{
		{
			Description:    "Success",
			MockGetItem:    MockOrderItemReturn{Item: someOrderItem},
			MockGetProduct: &MockProductReturn{Product: productWithStock},
			MockLockErr:    ptrErr(nil),
			MockUpdateErr:  ptrErr(nil),
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
			Description:    "Cart not draft / not owned (conditional update miss)",
			MockGetItem:    MockOrderItemReturn{Item: someOrderItem},
			MockGetProduct: &MockProductReturn{Product: productWithStock},
			MockLockErr:    ptrErr(nil),
			MockUpdateErr:  ptrErr(service.ErrNotFound),
			ExpectedErr:    service.ErrOrderStatusInvalid,
		},
		{
			Description:    "UpdateOrderItemQtyIfDraft fails",
			MockGetItem:    MockOrderItemReturn{Item: someOrderItem},
			MockGetProduct: &MockProductReturn{Product: productWithStock},
			MockLockErr:    ptrErr(nil),
			MockUpdateErr:  ptrErr(errors.New("db error")),
			ExpectedErr:    service.ErrUpdateOrderItem,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc, orderMock, itemMock, productMock, _, _ := newOrderService(ctrl)

			lockErr := error(nil)
			if tCase.MockLockErr != nil {
				lockErr = *tCase.MockLockErr
			}
			orderMock.EXPECT().LockUserCart(ctx, someID).Return(lockErr)

			itemMock.EXPECT().GetOrderItemByID(ctx, someItemID).Return(tCase.MockGetItem.Item, tCase.MockGetItem.Error)

			if tCase.MockGetProduct != nil {
				productMock.EXPECT().GetProductByID(ctx, someOrderItem.ProductID).Return(tCase.MockGetProduct.Product, tCase.MockGetProduct.Error)
			}
			if tCase.MockUpdateErr != nil {
				itemMock.EXPECT().UpdateOrderItemQtyIfDraft(ctx, someItemID, someID, newQuantity).Return(*tCase.MockUpdateErr)
			}

			err := svc.ChangeQuantityCartItem(ctx, testActor(someID, m.RoleBuyer), someItemID, newQuantity)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

func TestOrderService_DeleteCartItem(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		Description   string
		MockLockErr   *error
		MockDeleteErr *error
		ExpectedErr   error
	}

	tCases := []testCase{
		{
			Description:   "Success",
			MockLockErr:   ptrErr(nil),
			MockDeleteErr: ptrErr(nil),
		},
		{
			Description:   "Cart not draft / item not owned (conditional delete miss)",
			MockLockErr:   ptrErr(nil),
			MockDeleteErr: ptrErr(service.ErrNotFound),
			ExpectedErr:   service.ErrOrderItemNotFound,
		},
		{
			Description:   "Delete repo error",
			MockLockErr:   ptrErr(nil),
			MockDeleteErr: ptrErr(errors.New("db error")),
			ExpectedErr:   service.ErrDeleteOrderItemByID,
		},
		{
			Description: "Lock fails",
			MockLockErr: ptrErr(errors.New("db error")),
			ExpectedErr: service.ErrUpdateOrder,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc, orderMock, itemMock, _, _, _ := newOrderService(ctrl)

			if tCase.MockLockErr != nil {
				orderMock.EXPECT().LockUserCart(ctx, someID).Return(*tCase.MockLockErr)
			}
			if tCase.MockDeleteErr != nil {
				itemMock.EXPECT().DeleteOrderItemIfDraft(ctx, someItemID, someID).Return(*tCase.MockDeleteErr)
			}

			err := svc.DeleteCartItem(ctx, testActor(someID, m.RoleBuyer), someItemID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

func TestOrderService_Checkout(t *testing.T) {
	ctx := context.Background()

	product1 := m.Product{ID: 200, SellerID: 10, Price: decimal.NewFromFloat(50.00), StockQuantity: 10}
	product2 := m.Product{ID: 201, SellerID: 20, Price: decimal.NewFromFloat(30.00), StockQuantity: 10}

	item1 := m.OrderItem{ID: 300, OrderID: someOrderID, ProductID: 200, Quantity: 2, PriceAtPurchase: product1.Price}
	item2 := m.OrderItem{ID: 301, OrderID: someOrderID, ProductID: 201, Quantity: 1, PriceAtPurchase: product2.Price}

	type testCase struct {
		Description         string
		MockAddress         *MockAddressReturn
		MockGetOrders       MockOrderListReturn
		MockClaimErr        *error // UpdateOrderStatus draft→pending inside tx
		MockGetItems        *MockOrderItemListReturn
		MockProductsForLock map[int64]MockProductReturn
		// single seller path
		MockUpdateItemErr *error
		MockUpdateOrder   *MockOrderReturn
		// multi seller path
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
			MockClaimErr:  ptrErr(nil),
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1}},
			MockProductsForLock: map[int64]MockProductReturn{
				200: {Product: product1},
			},
			MockUpdateItemErr: ptrErr(nil),
			MockUpdateOrder:   &MockOrderReturn{},
			ExpectedCount:     1,
		},
		{
			Description:   "Success two sellers splits orders",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockClaimErr:  ptrErr(nil),
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1, item2}},
			MockProductsForLock: map[int64]MockProductReturn{
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
			Description: "Address not found",
			MockAddress: &MockAddressReturn{
				Error: service.ErrNotFound,
			},
			ExpectedErr: service.ErrAddressNotFound,
		},
		{
			Description: "Address belongs to another user",
			MockAddress: &MockAddressReturn{
				Address: m.Address{ID: someAddressID, UserID: otherUserID},
			},
			ExpectedErr: service.ErrNotYourAddress,
		},
		{
			Description:   "GetOrders repo error",
			MockGetOrders: MockOrderListReturn{Error: errors.New("db error")},
			ExpectedErr:   service.ErrGetCart,
		},
		{
			Description:   "Concurrent checkout (cart already claimed)",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockClaimErr:  ptrErr(service.ErrNotFound),
			ExpectedErr:   service.ErrCartNotFound,
		},
		{
			Description:   "Empty cart",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockClaimErr:  ptrErr(nil),
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{}},
			ExpectedErr:   service.ErrEmptyCart,
		},
		{
			Description:   "GetItems repo error",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockClaimErr:  ptrErr(nil),
			MockGetItems:  &MockOrderItemListReturn{Error: errors.New("db error")},
			ExpectedErr:   service.ErrGetOrderItemsByOrderID,
		},
		{
			Description:   "Product not found during checkout",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockClaimErr:  ptrErr(nil),
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1}},
			MockProductsForLock: map[int64]MockProductReturn{
				200: {Error: service.ErrNotFound},
			},
			ExpectedErr: service.ErrProductNotFound,
		},
		{
			Description:   "UpdateOrder fails (single seller)",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockClaimErr:  ptrErr(nil),
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1}},
			MockProductsForLock: map[int64]MockProductReturn{
				200: {Product: product1},
			},
			MockUpdateItemErr: ptrErr(nil),
			MockUpdateOrder:   &MockOrderReturn{Error: errors.New("db error")},
			ExpectedErr:       service.ErrUpdateOrder,
		},
		{
			Description:   "CreateOrder fails (two sellers)",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockClaimErr:  ptrErr(nil),
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1, item2}},
			MockProductsForLock: map[int64]MockProductReturn{
				200: {Product: product1},
				201: {Product: product2},
			},
			MockCreateOrds: []MockCreateReturn{{Error: errors.New("db error")}},
			ExpectedErr:    service.ErrCreateOrder,
		},
		{
			Description:   "CreateOrderItem fails during checkout (two sellers)",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockClaimErr:  ptrErr(nil),
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1, item2}},
			MockProductsForLock: map[int64]MockProductReturn{
				200: {Product: product1},
				201: {Product: product2},
			},
			MockCreateOrds: []MockCreateReturn{{ID: 600}},
			MockCreateItem: ptrErr(errors.New("db error")),
			ExpectedErr:    service.ErrCreateOrderItem,
		},
		{
			Description:   "DeleteCart fails after orders created (two sellers)",
			MockGetOrders: MockOrderListReturn{Orders: []m.Order{someDraftOrder}},
			MockClaimErr:  ptrErr(nil),
			MockGetItems:  &MockOrderItemListReturn{Items: []m.OrderItem{item1, item2}},
			MockProductsForLock: map[int64]MockProductReturn{
				200: {Product: product1},
				201: {Product: product2},
			},
			MockCreateOrds: []MockCreateReturn{{ID: 600}, {ID: 601}},
			MockCreateItem: ptrErr(nil),
			MockDeleteCart: ptrErr(errors.New("db error")),
			ExpectedErr:    service.ErrDeleteOrderByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc, orderMock, itemMock, productMock, _, addressMock := newOrderService(ctrl)

			orderMock.EXPECT().LockUserCart(ctx, someID).Return(nil)
			addressReturn := MockAddressReturn{Address: m.Address{ID: someAddressID, UserID: someID}}
			if tCase.MockAddress != nil {
				addressReturn = *tCase.MockAddress
			}
			addressMock.EXPECT().GetAddressByID(ctx, someAddressID).Return(addressReturn.Address, addressReturn.Error)
			addressFailed := addressReturn.Error != nil || addressReturn.Address.UserID != someID
			if addressFailed {
				orderIDs, err := svc.Checkout(ctx, testActor(someID, m.RoleBuyer), someAddressID)
				assertError(t, err, tCase.ExpectedErr)
				if len(orderIDs) != 0 {
					t.Fatalf("expected no order IDs, got %d", len(orderIDs))
				}
				return
			}

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

			if tCase.MockClaimErr != nil {
				orderMock.EXPECT().UpdateOrderStatus(ctx, someDraftOrder.ID,
					[]m.OrderStatus{m.StatusDraft}, m.StatusPending).Return(*tCase.MockClaimErr)
			}
			if tCase.MockGetItems != nil {
				itemMock.EXPECT().GetOrderItemsByOrderID(ctx, someDraftOrder.ID).Return(tCase.MockGetItems.Items, tCase.MockGetItems.Error)
			}
			for pid, ret := range tCase.MockProductsForLock {
				productMock.EXPECT().GetProductByIDForUpdate(ctx, pid).Return(ret.Product, ret.Error)
				if ret.Error == nil {
					qty := 0
					for _, item := range tCase.MockGetItems.Items {
						if item.ProductID == pid {
							qty += item.Quantity
						}
					}
					productMock.EXPECT().ChangeStockAndReserved(ctx, pid, 0, qty).Return(nil)
				}
			}
			if tCase.MockUpdateItemErr != nil {
				itemMock.EXPECT().UpdateOrderItem(ctx, gomock.Any(), gomock.Any()).Return(m.OrderItem{}, *tCase.MockUpdateItemErr).AnyTimes()
			}
			if tCase.MockUpdateOrder != nil {
				orderMock.EXPECT().UpdateOrder(ctx, someDraftOrder.ID, gomock.Any()).Return(tCase.MockUpdateOrder.Order, tCase.MockUpdateOrder.Error)
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

			orderIDs, err := svc.Checkout(ctx, testActor(someID, m.RoleBuyer), someAddressID)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && len(orderIDs) != tCase.ExpectedCount {
				t.Fatalf("expected %d order IDs, got %d", tCase.ExpectedCount, len(orderIDs))
			}
		})
	}
}

func TestOrderService_PayOrder(t *testing.T) {
	t.Run("admin is forbidden", func(t *testing.T) {
		svc, _, _, _, _, _ := newOrderService(gomock.NewController(t))

		err := svc.PayOrder(context.Background(), testActor(someID, m.RoleAdmin), someOrderID)

		assertError(t, err, service.ErrPermissionDenied)
	})
	ctx := context.Background()

	pendingOrder := someOrder // StatusPending, UserID = someID
	paidOrder := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusPaid}
	otherUsersOrder := m.Order{ID: someOrderID, UserID: otherUserID, Status: m.StatusPending}

	type testCase struct {
		Description      string
		MockGet          MockOrderReturn
		MockUpdateStatus *error
		ExpectedErr      error
	}

	tCases := []testCase{
		{
			Description:      "Success",
			MockGet:          MockOrderReturn{Order: pendingOrder},
			MockUpdateStatus: ptrErr(nil),
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
			Description:      "UpdateOrderStatus fails",
			MockGet:          MockOrderReturn{Order: pendingOrder},
			MockUpdateStatus: ptrErr(errors.New("db error")),
			ExpectedErr:      service.ErrUpdateOrder,
		},
		{
			Description:      "Concurrent pay — status already changed",
			MockGet:          MockOrderReturn{Order: pendingOrder},
			MockUpdateStatus: ptrErr(service.ErrNotFound),
			ExpectedErr:      service.ErrOrderStatusInvalid,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc, orderMock, _, _, _, _ := newOrderService(ctrl)

			orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(tCase.MockGet.Order, tCase.MockGet.Error)

			if tCase.MockUpdateStatus != nil {
				orderMock.EXPECT().UpdateOrderStatus(ctx, someOrderID, []m.OrderStatus{m.StatusPending}, m.StatusPaid).
					Return(*tCase.MockUpdateStatus)
			}

			err := svc.PayOrder(ctx, testActor(someID, m.RoleBuyer), someOrderID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

func TestOrderService_CancelOrder(t *testing.T) {
	t.Run("seller is forbidden", func(t *testing.T) {
		svc, _, _, _, _, _ := newOrderService(gomock.NewController(t))

		err := svc.CancelOrder(context.Background(), testActor(someID, m.RoleSeller), someOrderID)

		assertError(t, err, service.ErrPermissionDenied)
	})
	ctx := context.Background()

	draftOrder := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusDraft}
	pendingOrder := someOrder
	paidOrder := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusPaid}
	shippedOrder := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusShipped}
	deliveredOrder := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusDelivered}
	cancelledOrder := m.Order{ID: someOrderID, UserID: someID, Status: m.StatusCancelled}
	otherUsersOrder := m.Order{ID: someOrderID, UserID: otherUserID, Status: m.StatusPending}
	cancelItems := []m.OrderItem{
		{ID: 1, OrderID: someOrderID, ProductID: someProductID, Quantity: 2},
	}

	type testCase struct {
		Description      string
		MockGet          MockOrderReturn
		MockGetItems     bool
		MockUpdateStatus *error // UpdateOrderStatus result
		MockLock         bool   // GetProductByIDForUpdate called after status CAS
		MockRelease      bool
		ExpectedErr      error
	}

	tCases := []testCase{
		{
			Description:      "Success cancel pending",
			MockGet:          MockOrderReturn{Order: pendingOrder},
			MockGetItems:     true,
			MockUpdateStatus: ptrErr(nil),
			MockLock:         true,
			MockRelease:      true,
		},
		{
			Description:      "Success cancel paid",
			MockGet:          MockOrderReturn{Order: paidOrder},
			MockGetItems:     true,
			MockUpdateStatus: ptrErr(nil),
			MockLock:         true,
			MockRelease:      true,
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
			Description: "Cannot cancel draft",
			MockGet:     MockOrderReturn{Order: draftOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
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
			Description:      "UpdateOrderStatus fails",
			MockGet:          MockOrderReturn{Order: pendingOrder},
			MockGetItems:     true,
			MockUpdateStatus: ptrErr(errors.New("db error")),
			ExpectedErr:      service.ErrUpdateOrder,
		},
		{
			Description:      "Concurrent cancel — status already changed",
			MockGet:          MockOrderReturn{Order: pendingOrder},
			MockGetItems:     true,
			MockUpdateStatus: ptrErr(service.ErrNotFound),
			ExpectedErr:      service.ErrOrderStatusInvalid,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc, orderMock, itemMock, productMock, _, _ := newOrderService(ctrl)

			calls := []any{orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(tCase.MockGet.Order, tCase.MockGet.Error)}

			if tCase.MockGetItems {
				calls = append(calls, itemMock.EXPECT().GetOrderItemsByOrderID(ctx, someOrderID).Return(cancelItems, nil))
			}
			if tCase.MockUpdateStatus != nil {
				calls = append(calls, orderMock.EXPECT().UpdateOrderStatus(ctx, someOrderID,
					[]m.OrderStatus{m.StatusPending, m.StatusPaid}, m.StatusCancelled).Return(*tCase.MockUpdateStatus))
			}
			if tCase.MockLock {
				calls = append(calls, productMock.EXPECT().GetProductByIDForUpdate(ctx, someProductID).Return(m.Product{}, nil))
			}
			if tCase.MockRelease {
				calls = append(calls, productMock.EXPECT().ChangeStockAndReserved(ctx, someProductID, 0, -cancelItems[0].Quantity).Return(nil))
			}
			gomock.InOrder(calls...)

			err := svc.CancelOrder(ctx, testActor(someID, m.RoleBuyer), someOrderID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

type txManagerFunc func(context.Context, func(context.Context) error) error

func (f txManagerFunc) WithTransaction(ctx context.Context, callback func(context.Context) error) error {
	return f(ctx, callback)
}

func TestOrderService_ExpireOrders(t *testing.T) {
	ctx := context.Background()
	deadline := someTime.Add(-15 * time.Minute)
	order := someOrder
	smallID, largeID := int64(10), int64(30)
	products := map[int64]m.Product{
		smallID: newProductBuilder().withID(smallID).withStock(10, 3).build(),
		largeID: newProductBuilder().withID(largeID).withStock(10, 5).build(),
	}
	items := []m.OrderItem{
		{ID: 1, OrderID: order.ID, ProductID: largeID, Quantity: 2},
		{ID: 2, OrderID: order.ID, ProductID: smallID, Quantity: 3},
	}
	tests := []struct {
		name         string
		empty        bool
		selectionErr error
		itemsErr     error
		statusErr    error
		lockErr      error
		releaseErr   error
		wantErr      error
		wantDetails  *service.ExpireOrderError
	}{
		{name: "no expired orders", empty: true},
		{name: "cancels order and releases reservations after sorted locks"},
		{name: "concurrent payment skips order", statusErr: service.ErrNotFound},
		{name: "selection failure", selectionErr: errors.New("query failed"), wantErr: service.ErrGetOrdersByUserID},
		{name: "item lookup failure", itemsErr: errors.New("query failed"), wantErr: service.ErrGetOrderItemsByOrderID, wantDetails: &service.ExpireOrderError{OrderID: order.ID, Stage: service.ExpireStageGetItems}},
		{name: "status update failure", statusErr: errors.New("update failed"), wantErr: service.ErrUpdateOrder, wantDetails: &service.ExpireOrderError{OrderID: order.ID, Stage: service.ExpireStageUpdateStatus}},
		{name: "locked product not found", lockErr: service.ErrNotFound, wantErr: service.ErrProductNotFound, wantDetails: &service.ExpireOrderError{OrderID: order.ID, ProductID: smallID, Stage: service.ExpireStageLockProduct}},
		{name: "product lock failure", lockErr: errors.New("lock failed"), wantErr: service.ErrGetProductByID, wantDetails: &service.ExpireOrderError{OrderID: order.ID, ProductID: smallID, Stage: service.ExpireStageLockProduct}},
		{name: "release product not found", releaseErr: service.ErrNotFound, wantErr: service.ErrProductNotFound, wantDetails: &service.ExpireOrderError{OrderID: order.ID, ProductID: largeID, Quantity: 2, ReservedQuantity: 5, Stage: service.ExpireStageReleaseStock}},
		{name: "reservation invariant failure", releaseErr: service.ErrStockInvariantViolated, wantErr: service.ErrInsufficientStock, wantDetails: &service.ExpireOrderError{OrderID: order.ID, ProductID: largeID, Quantity: 2, ReservedQuantity: 5, Stage: service.ExpireStageReleaseStock}},
		{name: "reservation release failure", releaseErr: errors.New("update failed"), wantErr: service.ErrChangeStockAndReserved, wantDetails: &service.ExpireOrderError{OrderID: order.ID, ProductID: largeID, Quantity: 2, ReservedQuantity: 5, Stage: service.ExpireStageReleaseStock}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, orders, orderItems, productRepo, _, _ := newOrderService(gomock.NewController(t))
			found := []m.Order{order}
			if tc.empty {
				found = nil
			}
			calls := []any{orders.EXPECT().GetExpiredPendingOrders(ctx, deadline).Return(found, tc.selectionErr)}
			if !tc.empty && tc.selectionErr == nil {
				calls = append(calls, orderItems.EXPECT().GetOrderItemsByOrderID(ctx, order.ID).Return(items, tc.itemsErr))
				if tc.itemsErr == nil {
					calls = append(calls, orders.EXPECT().UpdateOrderStatus(ctx, order.ID, []m.OrderStatus{m.StatusPending}, m.StatusCancelled).Return(tc.statusErr))
					if tc.statusErr == nil {
						calls = append(calls, productRepo.EXPECT().GetProductByIDForUpdate(ctx, smallID).Return(products[smallID], tc.lockErr))
						if tc.lockErr == nil {
							calls = append(calls, productRepo.EXPECT().GetProductByIDForUpdate(ctx, largeID).Return(products[largeID], nil))
							calls = append(calls, productRepo.EXPECT().ChangeStockAndReserved(ctx, largeID, 0, -2).Return(tc.releaseErr))
							if tc.releaseErr == nil {
								calls = append(calls, productRepo.EXPECT().ChangeStockAndReserved(ctx, smallID, 0, -3).Return(nil))
							}
						}
					}
				}
			}
			gomock.InOrder(calls...)

			err := svc.ExpireOrders(ctx, deadline)

			assertError(t, err, tc.wantErr)
			if tc.wantDetails != nil {
				var details *service.ExpireOrderError
				if !errors.As(err, &details) {
					t.Fatalf("expected structured expiration error, got %v", err)
				}
				got, want := *details, *tc.wantDetails
				got.Cause = nil
				if got != want {
					t.Fatalf("unexpected error details: got %+v, want %+v", got, want)
				}
			}
		})
	}
	t.Run("transaction failure retains order context", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		orders := mock_service.NewMockOrderRepo(ctrl)
		orderItems := mock_service.NewMockOrderItemRepo(ctrl)
		failure := errors.New("transaction unavailable")
		tx := txManagerFunc(func(context.Context, func(context.Context) error) error { return failure })
		svc := service.NewOrderService(orders, orderItems, mock_service.NewMockProductRepo(ctrl), mock_service.NewMockAddressRepo(ctrl), mock_service.NewMockSellerGetter(ctrl), tx)
		orders.EXPECT().GetExpiredPendingOrders(ctx, deadline).Return([]m.Order{order}, nil)
		orderItems.EXPECT().GetOrderItemsByOrderID(ctx, order.ID).Return(items, nil)

		err := svc.ExpireOrders(ctx, deadline)

		assertError(t, err, failure)
		var details *service.ExpireOrderError
		if !errors.As(err, &details) || details.OrderID != order.ID || details.Stage != "tx" {
			t.Fatalf("unexpected transaction error context: %+v", details)
		}
	})
	t.Run("continues after failures and joins errors", func(t *testing.T) {
		svc, orders, orderItems, productRepo, _, _ := newOrderService(gomock.NewController(t))
		failedItemsOrder, failedStatusOrder, successfulOrder := someOrder, someOrder, someOrder
		failedItemsOrder.ID, failedStatusOrder.ID, successfulOrder.ID = 1, 2, 3
		gomock.InOrder(
			orders.EXPECT().GetExpiredPendingOrders(ctx, deadline).Return([]m.Order{failedItemsOrder, failedStatusOrder, successfulOrder}, nil),
			orderItems.EXPECT().GetOrderItemsByOrderID(ctx, failedItemsOrder.ID).Return(nil, errors.New("items unavailable")),
			orderItems.EXPECT().GetOrderItemsByOrderID(ctx, failedStatusOrder.ID).Return(items[:1], nil),
			orders.EXPECT().UpdateOrderStatus(ctx, failedStatusOrder.ID, []m.OrderStatus{m.StatusPending}, m.StatusCancelled).Return(errors.New("status unavailable")),
			orderItems.EXPECT().GetOrderItemsByOrderID(ctx, successfulOrder.ID).Return(items[:1], nil),
			orders.EXPECT().UpdateOrderStatus(ctx, successfulOrder.ID, []m.OrderStatus{m.StatusPending}, m.StatusCancelled).Return(nil),
			productRepo.EXPECT().GetProductByIDForUpdate(ctx, largeID).Return(products[largeID], nil),
			productRepo.EXPECT().ChangeStockAndReserved(ctx, largeID, 0, -2).Return(nil),
		)

		err := svc.ExpireOrders(ctx, deadline)

		assertError(t, err, service.ErrGetOrderItemsByOrderID)
		assertError(t, err, service.ErrUpdateOrder)
		var joined interface{ Unwrap() []error }
		if !errors.As(err, &joined) || len(joined.Unwrap()) != 2 {
			t.Fatalf("expected two accumulated failures, got %v", err)
		}
		for i, failure := range joined.Unwrap() {
			var details *service.ExpireOrderError
			if !errors.As(failure, &details) || details.OrderID != int64(i+1) {
				t.Fatalf("unexpected failure context: %+v", details)
			}
		}
	})
}

func TestOrderService_ShipOrder(t *testing.T) {
	t.Run("buyer is forbidden", func(t *testing.T) {
		svc, _, _, _, _, _ := newOrderService(gomock.NewController(t))

		err := svc.ShipOrder(context.Background(), testActor(someID, m.RoleBuyer), someOrderID)

		assertError(t, err, service.ErrPermissionDenied)
	})
	ctx := context.Background()

	seller := m.Seller{ID: someSellerID, UserID: someID}
	draftOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusDraft}
	paidOrderWithSeller := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusPaid}
	pendingOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusPending}
	alreadyShippedOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusShipped}
	deliveredOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusDelivered}
	cancelledOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusCancelled}
	otherSellerID := int64(999)
	otherSellersOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &otherSellerID, Status: m.StatusPaid}
	orderNoSeller := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: nil, Status: m.StatusPaid}
	someOrderItems := []m.OrderItem{
		{ID: 1, OrderID: someOrderID, ProductID: someProductID, Quantity: 2, PriceAtPurchase: someProductPrice},
	}

	type testCase struct {
		Description      string
		MockSeller       *MockSellerReturn
		MockGet          *MockOrderReturn
		MockUpdateStatus *error
		MockItems        bool
		MockLock         bool // GetProductByIDForUpdate fires before ChangeStockAndReserved
		MockChangeStock  bool
		ExpectedErr      error
	}

	tCases := []testCase{
		{
			Description:      "Success",
			MockSeller:       &MockSellerReturn{Seller: seller},
			MockGet:          &MockOrderReturn{Order: paidOrderWithSeller},
			MockUpdateStatus: ptrErr(nil),
			MockItems:        true,
			MockLock:         true,
			MockChangeStock:  true,
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
			Description: "Wrong status - draft",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: draftOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
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
			Description: "Wrong status - cancelled",
			MockSeller:  &MockSellerReturn{Seller: seller},
			MockGet:     &MockOrderReturn{Order: cancelledOrder},
			ExpectedErr: service.ErrOrderStatusInvalid,
		},
		{
			Description:      "UpdateOrderStatus fails",
			MockSeller:       &MockSellerReturn{Seller: seller},
			MockGet:          &MockOrderReturn{Order: paidOrderWithSeller},
			MockUpdateStatus: ptrErr(errors.New("db error")),
			ExpectedErr:      service.ErrUpdateOrder,
		},
		{
			Description:      "Concurrent ship — status already changed",
			MockSeller:       &MockSellerReturn{Seller: seller},
			MockGet:          &MockOrderReturn{Order: paidOrderWithSeller},
			MockUpdateStatus: ptrErr(service.ErrNotFound),
			ExpectedErr:      service.ErrOrderStatusInvalid,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc, orderMock, itemMock, productMock, sellerMock, _ := newOrderService(ctrl)

			if tCase.MockSeller != nil {
				sellerMock.EXPECT().GetSellerByUserID(ctx, someID).Return(tCase.MockSeller.Seller, tCase.MockSeller.Error)
			}
			if tCase.MockGet != nil {
				orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(tCase.MockGet.Order, tCase.MockGet.Error)
			}
			if tCase.MockUpdateStatus != nil {
				orderMock.EXPECT().UpdateOrderStatus(ctx, someOrderID, []m.OrderStatus{m.StatusPaid}, m.StatusShipped).
					Return(*tCase.MockUpdateStatus)
			}
			if tCase.MockItems {
				itemMock.EXPECT().GetOrderItemsByOrderID(ctx, someOrderID).Return(someOrderItems, nil)
			}
			if tCase.MockLock {
				productMock.EXPECT().GetProductByIDForUpdate(ctx, someProductID).Return(m.Product{}, nil)
			}
			if tCase.MockChangeStock {
				productMock.EXPECT().ChangeStockAndReserved(ctx, someProductID, -someOrderItems[0].Quantity, -someOrderItems[0].Quantity).Return(nil)
			}

			err := svc.ShipOrder(ctx, testActor(someID, m.RoleSeller), someOrderID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

func TestOrderService_DeliverOrder(t *testing.T) {
	ctx := context.Background()

	seller := m.Seller{ID: someSellerID, UserID: someID}
	shippedOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusShipped}
	paidOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusPaid}
	pendingOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusPending}
	alreadyDeliveredOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &someSellerID, Status: m.StatusDelivered}
	otherSellerID := int64(999)
	otherSellersOrder := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: &otherSellerID, Status: m.StatusShipped}
	orderNoSeller := m.Order{ID: someOrderID, UserID: otherUserID, SellerID: nil, Status: m.StatusShipped}

	type testCase struct {
		Description      string
		MockSeller       *MockSellerReturn
		MockGet          *MockOrderReturn
		MockUpdateStatus *error
		ExpectedErr      error
	}

	tCases := []testCase{
		{
			Description:      "Success",
			MockSeller:       &MockSellerReturn{Seller: seller},
			MockGet:          &MockOrderReturn{Order: shippedOrder},
			MockUpdateStatus: ptrErr(nil),
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
			Description:      "UpdateOrderStatus fails",
			MockSeller:       &MockSellerReturn{Seller: seller},
			MockGet:          &MockOrderReturn{Order: shippedOrder},
			MockUpdateStatus: ptrErr(errors.New("db error")),
			ExpectedErr:      service.ErrUpdateOrder,
		},
		{
			Description:      "Concurrent deliver — status already changed",
			MockSeller:       &MockSellerReturn{Seller: seller},
			MockGet:          &MockOrderReturn{Order: shippedOrder},
			MockUpdateStatus: ptrErr(service.ErrNotFound),
			ExpectedErr:      service.ErrOrderStatusInvalid,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc, orderMock, _, _, sellerMock, _ := newOrderService(ctrl)

			if tCase.MockSeller != nil {
				sellerMock.EXPECT().GetSellerByUserID(ctx, someID).Return(tCase.MockSeller.Seller, tCase.MockSeller.Error)
			}
			if tCase.MockGet != nil {
				orderMock.EXPECT().GetOrderByID(ctx, someOrderID).Return(tCase.MockGet.Order, tCase.MockGet.Error)
			}
			if tCase.MockUpdateStatus != nil {
				orderMock.EXPECT().UpdateOrderStatus(ctx, someOrderID, []m.OrderStatus{m.StatusShipped}, m.StatusDelivered).
					Return(*tCase.MockUpdateStatus)
			}

			err := svc.DeliverOrder(ctx, testActor(someID, m.RoleSeller), someOrderID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}
