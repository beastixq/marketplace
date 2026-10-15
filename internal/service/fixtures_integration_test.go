//go:build integration

package service_test

import (
	"context"
	"testing"
	"time"

	repo "github.com/beastixq/marketplace/internal/repository"
	s "github.com/beastixq/marketplace/internal/service"
	"github.com/beastixq/marketplace/internal/testsupport"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type serviceIntegration struct {
	t               *testing.T
	pool            *pgxpool.Pool
	ownedUserEmails []string
}

func newServiceIntegration(t *testing.T) *serviceIntegration {
	t.Helper()
	pool, err := testsupport.OpenIntegrationPostgres()
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	if pool == nil {
		t.Skip("set DATABASE_URL to run PostgreSQL integration tests")
	}
	f := &serviceIntegration{t: t, pool: pool}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		defer pool.Close()
		// All buyers' orders must be removed before any seller's products.
		for _, query := range []string{
			"DELETE FROM orders WHERE user_id IN (SELECT id FROM users WHERE email = ANY($1))",
			"DELETE FROM reviews WHERE user_id IN (SELECT id FROM users WHERE email = ANY($1))",
			"DELETE FROM products WHERE seller_id IN (SELECT id FROM sellers WHERE user_id IN (SELECT id FROM users WHERE email = ANY($1)))",
			"DELETE FROM sellers WHERE user_id IN (SELECT id FROM users WHERE email = ANY($1))",
			"DELETE FROM users WHERE email = ANY($1)",
		} {
			if _, err := pool.Exec(ctx, query, f.ownedUserEmails); err != nil {
				t.Errorf("clean owned integration data: %v", err)
			}
		}
	})
	return f
}

func (f *serviceIntegration) userService() s.UserService {
	return s.NewUserService(repo.NewUserRepo(f.pool), bcrypt.MinCost)
}

func (f *serviceIntegration) sellerService() s.SellerService {
	return s.NewSellerService(repo.NewSellerRepo(f.pool))
}

func (f *serviceIntegration) addressService() s.AddressService {
	return s.NewAddressService(repo.NewAddressRepo(f.pool))
}

func (f *serviceIntegration) productService() s.ProductService {
	return s.NewProductService(repo.NewProductRepo(f.pool), repo.NewReviewRepo(f.pool), repo.NewSellerRepo(f.pool), repo.NewPgxTxManager(f.pool))
}

func (f *serviceIntegration) orderService() s.OrderService {
	return s.NewOrderService(repo.NewOrderRepo(f.pool), repo.NewOrderItemRepo(f.pool), repo.NewProductRepo(f.pool), repo.NewAddressRepo(f.pool), repo.NewSellerRepo(f.pool), repo.NewPgxTxManager(f.pool))
}

func (f *serviceIntegration) reviewService() s.ReviewService {
	reviews := repo.NewReviewRepo(f.pool)
	return s.NewReviewService(reviews, reviews, repo.NewProductRepo(f.pool))
}

func mustIntegration(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
