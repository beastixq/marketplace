//go:build integration

package service_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	s "github.com/beastixq/marketplace/internal/service"
	"github.com/beastixq/marketplace/internal/testsupport"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

type integrationFixture struct {
	t           *testing.T
	pool        *pgxpool.Pool
	users       s.UserService
	sellers     s.SellerService
	addresses   s.AddressService
	products    s.ProductService
	orders      s.OrderService
	reviews     s.ReviewService
	userIDs     []int64
	categoryIDs []int64
}

func newIntegrationFixture(t *testing.T) *integrationFixture {
	t.Helper()
	pool, err := testsupport.OpenIntegrationPostgres()
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	if pool == nil {
		t.Skip("set DATABASE_URL to run PostgreSQL integration tests")
	}
	ur, sr := repo.NewUserRepo(pool), repo.NewSellerRepo(pool)
	ar, pr := repo.NewAddressRepo(pool), repo.NewProductRepo(pool)
	or, ir, rr := repo.NewOrderRepo(pool), repo.NewOrderItemRepo(pool), repo.NewReviewRepo(pool)
	tx := repo.NewPgxTxManager(pool)
	f := &integrationFixture{t: t, pool: pool, users: s.NewUserService(ur, bcrypt.MinCost),
		sellers: s.NewSellerService(sr), addresses: s.NewAddressService(ar),
		products: s.NewProductService(pr, rr, sr, tx), orders: s.NewOrderService(or, ir, pr, ar, sr, tx),
		reviews: s.NewReviewService(rr, rr, pr)}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		defer pool.Close()
		for _, query := range []string{
			"DELETE FROM orders WHERE user_id = ANY($1)",
			"DELETE FROM reviews WHERE user_id = ANY($1)",
			"DELETE FROM products WHERE seller_id IN (SELECT id FROM sellers WHERE user_id = ANY($1))",
			"DELETE FROM sellers WHERE user_id = ANY($1)",
			"DELETE FROM users WHERE id = ANY($1)",
		} {
			if _, err := pool.Exec(ctx, query, f.userIDs); err != nil {
				t.Errorf("clean owned fixture: %v", err)
			}
		}
		if _, err := pool.Exec(ctx, "DELETE FROM categories WHERE id = ANY($1)", f.categoryIDs); err != nil {
			t.Errorf("clean categories: %v", err)
		}
	})
	return f
}

func (f *integrationFixture) actor(role m.UserRole) s.Actor {
	f.t.Helper()
	id, err := f.users.CreateUser(context.Background(), m.UserCreate{
		Email:    fmt.Sprintf("integration-%d-%d@example.test", time.Now().UnixNano(), len(f.userIDs)),
		Password: "Synthetic-password-42", FullName: "Integration account", Role: role,
	})
	if err != nil {
		f.t.Fatalf("create account: %v", err)
	}
	f.userIDs = append(f.userIDs, id)
	if os.Getenv("TEST_FAULT_GATE") == "integration" {
		// Acceptance kills the container only after a committed fixture exists.
		mustIntegration(f.t, os.WriteFile("/run/integration-written", []byte("committed"), 0600))
		<-time.After(180 * time.Second)
		f.t.Fatal("acceptance did not terminate the integration container in 180 seconds")
	}
	return s.Actor{UserID: id, Role: role}
}

func (f *integrationFixture) seller() (s.Actor, int64) {
	f.t.Helper()
	actor := f.actor(m.RoleSeller)
	id, err := f.sellers.CreateSeller(context.Background(), actor, m.SellerCreate{UserID: actor.UserID, CompanyName: "Integration seller"})
	if err != nil {
		f.t.Fatalf("create seller: %v", err)
	}
	return actor, id
}

func (f *integrationFixture) product(actor s.Actor, sellerID int64, stock int) int64 {
	f.t.Helper()
	id, err := f.products.CreateProduct(context.Background(), actor, m.ProductCreate{
		SellerID: sellerID, Name: "Integration product", Price: decimal.NewFromInt(100), StockQuantity: stock,
	})
	if err != nil {
		f.t.Fatalf("create product: %v", err)
	}
	return id
}

func (f *integrationFixture) address(actor s.Actor) int64 {
	f.t.Helper()
	id, err := f.addresses.CreateAddress(context.Background(), actor, m.AddressCreate{
		UserID: actor.UserID, City: "Test city", Street: "Test street", House: "1", ZipCode: "123456",
	})
	if err != nil {
		f.t.Fatalf("create address: %v", err)
	}
	return id
}

func mustIntegration(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func (f *integrationFixture) checkout(buyer s.Actor, productID, addressID int64) int64 {
	f.t.Helper()
	ctx := context.Background()
	mustIntegration(f.t, f.orders.AddItemToCart(ctx, buyer, productID, 1))
	ids, err := f.orders.Checkout(ctx, buyer, addressID)
	mustIntegration(f.t, err)
	if len(ids) != 1 {
		f.t.Fatalf("orders = %v, want one", ids)
	}
	return ids[0]
}

func (f *integrationFixture) stock(productID int64, stock, reserved int) {
	f.t.Helper()
	p, err := f.products.GetProductByID(context.Background(), productID)
	mustIntegration(f.t, err)
	if p.StockQuantity != stock || p.ReservedQuantity != reserved {
		f.t.Fatalf("stock/reserved = %d/%d, want %d/%d", p.StockQuantity, p.ReservedQuantity, stock, reserved)
	}
}
