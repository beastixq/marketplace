package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beastixq/marketplace/cmd/seed/generators"
)

// type UsersGenerator interface {
// 	CreateUsers(tx pgxpool.tx, ctx, ctx context.Context, count int) (users []int64, err error);
// }

func main() {
	const (
		adminsCount     = 100
		analystsCount   = 100
		buyersCount     = 1000
		sellersCount    = 1000
		categoriesCount = 1000
		productsCount   = 5000
		addressesCount  = 2000
		ordersCount     = 1000
		orderItemsCount = 3000
		reviewsCount    = 2500
	)

	log.Println("Start seeding...")

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to create New pool: %v\n", err)
	}
	defer pool.Close()
	log.Println("Pool created!")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()

	rbCtx, rbCancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer rbCancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Printf("Failed to begin transaction: %v\n", err)
		return
	}
	log.Println("Begin transaction!")
	defer tx.Rollback(rbCtx)

	faker.ResetUnique()
	// Users
	log.Printf("Start creating %d users...\n", adminsCount+analystsCount+sellersCount+buyersCount)
	usersIDs, err := generators.CreateUsers(tx, ctx, adminsCount, analystsCount, buyersCount, sellersCount)
	if err != nil {
		log.Printf("Failed on CreateUsers: %v\n", err)
		return
	}
	log.Println("Users created: ", len(usersIDs))
	log.Println("First 10 usersSellers IDs:")
	log.Println(usersIDs[3][:10])
	// Sellers
	log.Printf("Start creating %d sellers...\n", sellersCount)
	sellersIDs, err := generators.CreateSellers(tx, ctx, usersIDs[3])
	if err != nil {
		log.Printf("Failed on CreateSellers: %v\n", err)
		return
	}
	log.Println("Sellers created: ", len(sellersIDs))
	log.Println("First 10 sellers IDs:")
	log.Println(sellersIDs[:10])
	// Categories
	log.Printf("Start creating %d categories...\n", categoriesCount)
	categoriesIDs, err := generators.CreateRealCategories(tx, ctx, categoriesCount)
	if err != nil {
		log.Printf("Failed on CreateCategories: %v\n", err)
		return
	}
	log.Println("Categories created: ", len(categoriesIDs))
	// Products
	log.Printf("Start creating %d products...\n", productsCount)
	productsIDs, err := generators.CreateProducts(tx, ctx, sellersIDs, productsCount)
	if err != nil {
		log.Printf("Failed on CreateProducts: %v\n", err)
		return
	}
	log.Println("Products created: ", len(productsIDs))
	// ProductCategories
	log.Println("Start creating product-category links...")
	err = generators.CreateProductCategories(tx, ctx, productsIDs, categoriesIDs)
	if err != nil {
		log.Printf("Failed on CreateProductCategories: %v\n", err)
		return
	}
	log.Println("ProductCategories created")
	// Addresses (for buyers)
	log.Printf("Start creating %d addresses...\n", addressesCount)
	addressesIDs, err := generators.CreateAddresses(tx, ctx, usersIDs[2], addressesCount)
	if err != nil {
		log.Printf("Failed on CreateAddresses: %v\n", err)
		return
	}
	log.Println("Addresses created: ", len(addressesIDs))
	// Orders (buyers place orders)
	log.Printf("Start creating %d orders...\n", ordersCount)
	ordersIDs, err := generators.CreateOrders(tx, ctx, usersIDs[2], addressesIDs, ordersCount)
	if err != nil {
		log.Printf("Failed on CreateOrders: %v\n", err)
		return
	}
	log.Println("Orders created: ", len(ordersIDs))
	// OrderItems
	log.Printf("Start creating %d order items...\n", orderItemsCount)
	err = generators.CreateOrderItems(tx, ctx, ordersIDs, productsIDs, orderItemsCount)
	if err != nil {
		log.Printf("Failed on CreateOrderItems: %v\n", err)
		return
	}
	log.Println("OrderItems created")
	// Reviews (buyers review products)
	log.Printf("Start creating %d reviews...\n", reviewsCount)
	err = generators.CreateReviews(tx, ctx, usersIDs[2], productsIDs, reviewsCount)
	if err != nil {
		log.Printf("Failed on CreateReviews: %v\n", err)
		return
	}
	log.Println("Reviews created")

	log.Println("All generators succeeded!")
	if err = tx.Commit(ctx); err != nil {
		log.Printf("Failed to commit: %v\n", err)
		return
	}
	log.Println("Commit succeeded!")
	log.Println("Seeding end")
}
