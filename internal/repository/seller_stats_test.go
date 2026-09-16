package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/shopspring/decimal"
)

// Классы эквивалентности для get_seller_statistics (см. РПЗ, таблица «Классы
// эквивалентности для тестирования функции get_seller_statistics»):
//
//	K1 — нет завершённых продаж в периоде;
//	K2 — один оплаченный заказ с одной позицией;
//	K3 — один заказ с несколькими позициями товаров продавца;
//	K4 — заказы со статусами draft/pending/cancelled исключаются;
//	K5 — заказы вне временного интервала не учитываются;
//	K6 — top_product_name определяется по максимальной сумме quantity;
//	K7 — при равенстве сумм quantity возвращается один из лидирующих товаров.
//
// Период тестов: [statsFrom, statsTo). Функция фильтрует заказы по
// created_at >= p_date_from AND created_at < p_date_to.
var (
	statsFrom    = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	statsTo      = time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	statsInside  = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	statsOutside = time.Date(2023, 12, 15, 12, 0, 0, 0, time.UTC)
)

// statsProduct создаёт товар продавца с заданным именем и регистрирует очистку.
func statsProduct(t *testing.T, sellerID int64, name string) int64 {
	t.Helper()
	ctx := context.Background()
	r := repo.NewProductRepo(testPool)
	id, err := r.CreateProduct(ctx, m.ProductCreate{
		SellerID:      sellerID,
		Name:          name,
		Price:         decimal.NewFromFloat(100),
		StockQuantity: 1000,
	})
	if err != nil {
		t.Fatalf("statsProduct(%q): %v", name, err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM product_price_history WHERE product_id = $1", id)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", id)
	})
	return id
}

type statsItem struct {
	productID int64
	quantity  int
	price     float64
}

// statsOrder создаёт заказ продавца в заданном статусе с позициями и
// проставляет created_at. Возвращает ID заказа.
//
// Удаление заказа в очистке каскадит на order_items (ON DELETE CASCADE),
// поэтому очистка товаров (RESTRICT по product_id) проходит после неё.
func statsOrder(t *testing.T, buyerID, sellerID int64, status m.OrderStatus, createdAt time.Time, items ...statsItem) int64 {
	t.Helper()
	ctx := context.Background()
	orderRepo := repo.NewOrderRepo(testPool)
	itemRepo := repo.NewOrderItemRepo(testPool)

	orderID, err := orderRepo.CreateOrder(ctx, m.OrderCreate{
		UserID:      buyerID,
		SellerID:    &sellerID,
		Status:      status,
		TotalAmount: decimal.NewFromFloat(0),
	})
	if err != nil {
		t.Fatalf("statsOrder: CreateOrder: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", orderID)
	})

	for _, it := range items {
		if _, err := itemRepo.CreateOrderItem(ctx, m.OrderItemCreate{
			OrderID:         orderID,
			ProductID:       it.productID,
			Quantity:        it.quantity,
			PriceAtPurchase: decimal.NewFromFloat(it.price),
		}); err != nil {
			t.Fatalf("statsOrder: CreateOrderItem: %v", err)
		}
	}

	if _, err := testPool.Exec(ctx, "UPDATE orders SET created_at = $1 WHERE id = $2", createdAt, orderID); err != nil {
		t.Fatalf("statsOrder: set created_at: %v", err)
	}
	return orderID
}

// K1: у продавца нет завершённых продаж в периоде.
// Агрегаты выручки, среднего чека и лидирующего товара формируются как NULL
// (count(distinct) = 0, sum = NULL, подзапрос top без строк = NULL).
// Текущая реализация репозитория сканирует их в ненулевые типы
// (decimal.Decimal, string), поэтому GetSellerStats возвращает ошибку scan.
func TestGetSellerStats_K1_NoCompletedSales(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	r := repo.NewSellerRepo(testPool)
	ctx := context.Background()

	// Отменённый заказ в периоде не создаёт завершённых продаж.
	prod := statsProduct(t, sellerID, "K1 product")
	statsOrder(t, buyerID, sellerID, m.StatusCancelled, statsInside, statsItem{prod, 1, 100})

	_, err := r.GetSellerStats(ctx, sellerID, statsFrom, statsTo)
	if !errors.Is(err, repo.ErrToScan) {
		t.Fatalf("K1: ожидалась ошибка scan (NULL-агрегаты), получено: %v", err)
	}
}

// K2: один оплаченный заказ с одной позицией.
func TestGetSellerStats_K2_SingleOrderSingleItem(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	r := repo.NewSellerRepo(testPool)
	ctx := context.Background()

	prod := statsProduct(t, sellerID, "K2 product")
	statsOrder(t, buyerID, sellerID, m.StatusPaid, statsInside, statsItem{prod, 2, 50})

	got, err := r.GetSellerStats(ctx, sellerID, statsFrom, statsTo)
	if err != nil {
		t.Fatalf("K2: GetSellerStats: %v", err)
	}
	if got.TotalOrders != 1 {
		t.Errorf("K2: TotalOrders = %d, want 1", got.TotalOrders)
	}
	if want := decimal.NewFromInt(100); !got.TotalRevenue.Equal(want) {
		t.Errorf("K2: TotalRevenue = %s, want %s", got.TotalRevenue, want)
	}
	if want := decimal.NewFromInt(100); !got.AvgOrderValue.Equal(want) {
		t.Errorf("K2: AvgOrderValue = %s, want %s", got.AvgOrderValue, want)
	}
	if got.TopProductName != "K2 product" {
		t.Errorf("K2: TopProductName = %q, want %q", got.TopProductName, "K2 product")
	}
}

// K3: один заказ содержит несколько позиций товаров продавца — заказ
// учитывается один раз, выручка суммируется по всем позициям.
func TestGetSellerStats_K3_SingleOrderMultipleItems(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	r := repo.NewSellerRepo(testPool)
	ctx := context.Background()

	p1 := statsProduct(t, sellerID, "K3 product A")
	p2 := statsProduct(t, sellerID, "K3 product B")
	statsOrder(t, buyerID, sellerID, m.StatusPaid, statsInside,
		statsItem{p1, 1, 30},
		statsItem{p2, 1, 70},
	)

	got, err := r.GetSellerStats(ctx, sellerID, statsFrom, statsTo)
	if err != nil {
		t.Fatalf("K3: GetSellerStats: %v", err)
	}
	if got.TotalOrders != 1 {
		t.Errorf("K3: TotalOrders = %d, want 1 (заказ учитывается один раз)", got.TotalOrders)
	}
	if want := decimal.NewFromInt(100); !got.TotalRevenue.Equal(want) {
		t.Errorf("K3: TotalRevenue = %s, want %s", got.TotalRevenue, want)
	}
}

// K4: заказы со статусами draft/pending/cancelled исключаются из расчёта.
func TestGetSellerStats_K4_NonCompletedStatusesExcluded(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	r := repo.NewSellerRepo(testPool)
	ctx := context.Background()

	prod := statsProduct(t, sellerID, "K4 product")
	// Один завершённый заказ как базовая линия.
	statsOrder(t, buyerID, sellerID, m.StatusPaid, statsInside, statsItem{prod, 1, 100})
	// Заказы, которые должны быть исключены.
	statsOrder(t, buyerID, sellerID, m.StatusDraft, statsInside, statsItem{prod, 5, 100})
	statsOrder(t, buyerID, sellerID, m.StatusPending, statsInside, statsItem{prod, 5, 100})
	statsOrder(t, buyerID, sellerID, m.StatusCancelled, statsInside, statsItem{prod, 5, 100})

	got, err := r.GetSellerStats(ctx, sellerID, statsFrom, statsTo)
	if err != nil {
		t.Fatalf("K4: GetSellerStats: %v", err)
	}
	if got.TotalOrders != 1 {
		t.Errorf("K4: TotalOrders = %d, want 1 (учитывается только paid)", got.TotalOrders)
	}
	if want := decimal.NewFromInt(100); !got.TotalRevenue.Equal(want) {
		t.Errorf("K4: TotalRevenue = %s, want %s (исключённые заказы не влияют)", got.TotalRevenue, want)
	}
}

// K5: заказы вне заданного временного интервала не влияют на агрегаты.
func TestGetSellerStats_K5_OutOfRangeExcluded(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	r := repo.NewSellerRepo(testPool)
	ctx := context.Background()

	prod := statsProduct(t, sellerID, "K5 product")
	statsOrder(t, buyerID, sellerID, m.StatusPaid, statsInside, statsItem{prod, 1, 100})  // в периоде
	statsOrder(t, buyerID, sellerID, m.StatusPaid, statsOutside, statsItem{prod, 9, 100}) // вне периода

	got, err := r.GetSellerStats(ctx, sellerID, statsFrom, statsTo)
	if err != nil {
		t.Fatalf("K5: GetSellerStats: %v", err)
	}
	if got.TotalOrders != 1 {
		t.Errorf("K5: TotalOrders = %d, want 1 (заказ вне интервала не учитывается)", got.TotalOrders)
	}
	if want := decimal.NewFromInt(100); !got.TotalRevenue.Equal(want) {
		t.Errorf("K5: TotalRevenue = %s, want %s", got.TotalRevenue, want)
	}
}

// K6: продано несколько товаров с разным количеством единиц —
// top_product_name соответствует товару с максимальной суммой quantity.
func TestGetSellerStats_K6_TopProductByQuantity(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	r := repo.NewSellerRepo(testPool)
	ctx := context.Background()

	leader := statsProduct(t, sellerID, "K6 leader")
	other := statsProduct(t, sellerID, "K6 other")
	// leader: суммарно 7 единиц (4 + 3); other: 2 единицы.
	statsOrder(t, buyerID, sellerID, m.StatusPaid, statsInside,
		statsItem{leader, 4, 10},
		statsItem{other, 2, 10},
	)
	statsOrder(t, buyerID, sellerID, m.StatusDelivered, statsInside,
		statsItem{leader, 3, 10},
	)

	got, err := r.GetSellerStats(ctx, sellerID, statsFrom, statsTo)
	if err != nil {
		t.Fatalf("K6: GetSellerStats: %v", err)
	}
	if got.TopProductName != "K6 leader" {
		t.Errorf("K6: TopProductName = %q, want %q", got.TopProductName, "K6 leader")
	}
}

// K7: два товара имеют одинаковую максимальную суммарную проданную величину —
// возвращается один из лидирующих товаров (конкретный выбор не доопределён).
func TestGetSellerStats_K7_TopProductTie(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	r := repo.NewSellerRepo(testPool)
	ctx := context.Background()

	a := statsProduct(t, sellerID, "K7 product A")
	b := statsProduct(t, sellerID, "K7 product B")
	// Оба товара проданы в равном количестве — 5 единиц.
	statsOrder(t, buyerID, sellerID, m.StatusPaid, statsInside,
		statsItem{a, 5, 10},
		statsItem{b, 5, 10},
	)

	got, err := r.GetSellerStats(ctx, sellerID, statsFrom, statsTo)
	if err != nil {
		t.Fatalf("K7: GetSellerStats: %v", err)
	}
	if got.TopProductName != "K7 product A" && got.TopProductName != "K7 product B" {
		t.Errorf("K7: TopProductName = %q, want один из {%q, %q}", got.TopProductName, "K7 product A", "K7 product B")
	}
}
