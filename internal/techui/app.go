package techui

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/shopspring/decimal"
)

type AuthSession interface {
	Register(ctx context.Context, uc m.UserCreate) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
	Logout(ctx context.Context, claims m.TokenClaims) error
	ValidateToken(ctx context.Context, token string) (m.TokenClaims, error)
}

type UserProfile interface {
	GetUserByID(ctx context.Context, id int64) (m.User, error)
	UpdateUser(ctx context.Context, id int64, uu m.UserUpdate) (m.User, error)
	ChangePasswordUser(ctx context.Context, id int64, oldPass, newPass string) error
	DeleteUserByID(ctx context.Context, id int64) error
}

type UserAdministration interface {
	GetUsers(ctx context.Context, opts m.UserListOptions) ([]m.User, error)
	GetUserByID(ctx context.Context, id int64) (m.User, error)
	UpdateUser(ctx context.Context, id int64, uu m.UserUpdate) (m.User, error)
	DeleteUserByID(ctx context.Context, id int64) error
}

type SellerProfile interface {
	GetSellerByID(ctx context.Context, id int64) (m.Seller, error)
	GetSellerByUserID(ctx context.Context, userID int64) (m.Seller, error)
	CreateSeller(ctx context.Context, sc m.SellerCreate) (int64, error)
	UpdateSeller(ctx context.Context, userID, id int64, su m.SellerUpdate) (m.Seller, error)
	DeleteSellerByID(ctx context.Context, userID, id int64) error
}

type SellerStatistics interface {
	GetSellerStats(ctx context.Context, userID, sellerID int64, dateFrom, dateTo time.Time) (m.SellerStats, error)
}

type ProductCatalog interface {
	GetProducts(ctx context.Context, options m.CatalogOptions) ([]m.Product, error)
}

type ProductDetails interface {
	GetProductByID(ctx context.Context, id int64) (m.Product, error)
	GetProductPriceHistory(ctx context.Context, pid int64, dateFrom time.Time, dateTo time.Time) ([]m.ProductPriceHistory, error)
	GetReviewsByProductID(ctx context.Context, pid int64, opts m.PaginationOpts) ([]m.Review, error)
}

type ProductManagement interface {
	CreateProduct(ctx context.Context, userID int64, pc m.ProductCreate) (int64, error)
	UpdateProduct(ctx context.Context, userID, id int64, pu m.ProductUpdate) (m.Product, error)
	DeleteProductByID(ctx context.Context, userID, id int64) error
}

type CategoryBrowser interface {
	GetCategories(ctx context.Context, opts m.PaginationOpts) ([]m.Category, error)
	GetCategoryByID(ctx context.Context, id int64) (m.Category, error)
}

type CategoryManagement interface {
	CreateCategory(ctx context.Context, cc m.CategoryCreate) (int64, error)
	UpdateCategory(ctx context.Context, id int64, cu m.CategoryUpdate) (m.Category, error)
	DeleteCategoryByID(ctx context.Context, id int64) error
}

type AddressBook interface {
	GetAddressesByUserID(ctx context.Context, userID int64) ([]m.Address, error)
	CreateAddress(ctx context.Context, ac m.AddressCreate) (int64, error)
	UpdateAddress(ctx context.Context, userID, id int64, au m.AddressUpdate) (m.Address, error)
	DeleteAddressByID(ctx context.Context, userID, id int64) error
}

type Cart interface {
	GetCart(ctx context.Context, userID int64) (m.Order, error)
	GetOrderItemsByOrderID(ctx context.Context, orderID int64) ([]m.OrderItem, error)
	AddItemToCart(ctx context.Context, userID int64, productID int64, quantity int) error
	ChangeQuantityCartItem(ctx context.Context, userID int64, itemID int64, quantity int) error
	DeleteCartItem(ctx context.Context, userID int64, itemID int64) error
	Checkout(ctx context.Context, userID int64, addressID int64) ([]int64, error)
}

type BuyerOrders interface {
	GetOrderByID(ctx context.Context, orderID int64) (m.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int64, pg m.PaginationOpts) ([]m.Order, error)
	GetOrderItemsByOrderID(ctx context.Context, orderID int64) ([]m.OrderItem, error)
	PayOrder(ctx context.Context, orderID int64, userID int64) error
	CancelOrder(ctx context.Context, orderID int64, userID int64) error
}

type SellerOrders interface {
	GetSellerOrdersByUserID(ctx context.Context, userID int64, pg m.PaginationOpts) ([]m.Order, error)
	ShipOrder(ctx context.Context, orderID int64, userID int64) error
	DeliverOrder(ctx context.Context, orderID int64, userID int64) error
}

type Payments interface {
	GetOrderPaymentURL(ctx context.Context, orderID, userID int64) (string, time.Time, error)
	ProcessOrderPayment(ctx context.Context, token string) error
}

type ReviewManagement interface {
	GetReviewByID(ctx context.Context, id int64) (m.Review, error)
	CreateReview(ctx context.Context, rc m.ReviewCreate) (int64, error)
	UpdateReview(ctx context.Context, userID, id int64, ru m.ReviewUpdate) (m.Review, error)
	DeleteReviewByID(ctx context.Context, userID, id int64) error
}

type ServicePorts struct {
	Auth               AuthSession
	UserProfile        UserProfile
	UserAdministration UserAdministration
	SellerProfile      SellerProfile
	SellerStatistics   SellerStatistics
	ProductCatalog     ProductCatalog
	ProductDetails     ProductDetails
	ProductManagement  ProductManagement
	CategoryBrowser    CategoryBrowser
	CategoryManagement CategoryManagement
	AddressBook        AddressBook
	Cart               Cart
	BuyerOrders        BuyerOrders
	SellerOrders       SellerOrders
	Payments           Payments
	ReviewManagement   ReviewManagement
}

type App struct {
	servicePorts ServicePorts
	in           *bufio.Reader
	out          io.Writer
	token        string
	claims       *m.TokenClaims
}

func New(servicePorts ServicePorts, in io.Reader, out io.Writer) *App {
	return &App{
		servicePorts: servicePorts,
		in:           bufio.NewReader(in),
		out:          out,
	}
}

func (a *App) Run(ctx context.Context) error {
	a.println("Marketplace technological UI")
	for {
		if a.claims == nil {
			if err := a.guestMenu(ctx); err != nil {
				return err
			}
			continue
		}
		if err := a.userMenu(ctx); err != nil {
			return err
		}
	}
}

func (a *App) guestMenu(ctx context.Context) error {
	a.println("")
	a.println("1. Register")
	a.println("2. Login")
	a.println("3. Products catalog")
	a.println("4. Product details")
	a.println("5. Categories")
	a.println("0. Exit")
	choice, err := a.readLine("Select")
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		a.run(ctx, a.register)
	case "2":
		a.run(ctx, a.login)
	case "3":
		a.run(ctx, a.listProducts)
	case "4":
		a.run(ctx, a.showProduct)
	case "5":
		a.run(ctx, a.listCategories)
	case "0":
		return io.EOF
	default:
		a.println("Unknown command")
	}
	return nil
}

func (a *App) userMenu(ctx context.Context) error {
	a.println("")
	a.printf("User: id=%d role=%s\n", a.claims.UserID, a.claims.Role)
	a.println("1. Profile")
	a.println("2. Products")
	a.println("3. Categories")
	a.println("4. Addresses")
	a.println("5. Cart and orders")
	a.println("6. Seller workspace")
	a.println("7. Reviews")
	a.println("8. Admin workspace")
	a.println("9. Logout")
	a.println("0. Exit")
	choice, err := a.readLine("Select")
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		a.run(ctx, a.profileMenu)
	case "2":
		a.run(ctx, a.productsMenu)
	case "3":
		a.run(ctx, a.categoriesMenu)
	case "4":
		a.run(ctx, a.addressesMenu)
	case "5":
		a.run(ctx, a.ordersMenu)
	case "6":
		a.run(ctx, a.sellerMenu)
	case "7":
		a.run(ctx, a.reviewsMenu)
	case "8":
		a.run(ctx, a.adminMenu)
	case "9":
		a.run(ctx, a.logout)
	case "0":
		return io.EOF
	default:
		a.println("Unknown command")
	}
	return nil
}

func (a *App) register(ctx context.Context) error {
	email, err := a.readRequired("Email")
	if err != nil {
		return err
	}
	password, err := a.readRequired("Password")
	if err != nil {
		return err
	}
	fullName, err := a.readRequired("Full name")
	if err != nil {
		return err
	}
	phone, err := a.readOptionalString("Phone")
	if err != nil {
		return err
	}
	role, err := a.readRole("Role [buyer/seller/admin/analyst]", m.RoleBuyer)
	if err != nil {
		return err
	}
	token, err := a.servicePorts.Auth.Register(ctx, m.UserCreate{
		Email:    email,
		Password: password,
		FullName: fullName,
		Phone:    phone,
		Role:     role,
	})
	if err != nil {
		return err
	}
	return a.setToken(ctx, token)
}

func (a *App) login(ctx context.Context) error {
	email, err := a.readRequired("Email")
	if err != nil {
		return err
	}
	password, err := a.readRequired("Password")
	if err != nil {
		return err
	}
	token, err := a.servicePorts.Auth.Login(ctx, email, password)
	if err != nil {
		return err
	}
	return a.setToken(ctx, token)
}

func (a *App) logout(ctx context.Context) error {
	if a.claims != nil {
		if err := a.servicePorts.Auth.Logout(ctx, *a.claims); err != nil {
			return err
		}
	}
	a.claims = nil
	a.token = ""
	a.println("Logged out")
	return nil
}

func (a *App) setToken(ctx context.Context, token string) error {
	claims, err := a.servicePorts.Auth.ValidateToken(ctx, token)
	if err != nil {
		return err
	}
	a.token = token
	a.claims = &claims
	a.printf("Authenticated: user_id=%d role=%s\n", claims.UserID, claims.Role)
	return nil
}

func (a *App) profileMenu(ctx context.Context) error {
	a.println("")
	a.println("1. Show profile")
	a.println("2. Update profile")
	a.println("3. Change password")
	a.println("4. Delete profile")
	a.println("0. Back")
	choice, err := a.readLine("Select")
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		return a.showProfile(ctx)
	case "2":
		return a.updateProfile(ctx)
	case "3":
		return a.changePassword(ctx)
	case "4":
		return a.deleteProfile(ctx)
	case "0":
		return nil
	default:
		a.println("Unknown command")
		return nil
	}
}

func (a *App) showProfile(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	user, err := a.servicePorts.UserProfile.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return err
	}
	a.printUser(user)
	return nil
}

func (a *App) updateProfile(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	email, err := a.readOptionalString("New email")
	if err != nil {
		return err
	}
	name, err := a.readOptionalString("New full name")
	if err != nil {
		return err
	}
	phone, err := a.readOptionalString("New phone")
	if err != nil {
		return err
	}
	user, err := a.servicePorts.UserProfile.UpdateUser(ctx, claims.UserID, m.UserUpdate{
		Email:    email,
		FullName: name,
		Phone:    phone,
	})
	if err != nil {
		return err
	}
	a.printUser(user)
	return nil
}

func (a *App) changePassword(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	oldPass, err := a.readRequired("Old password")
	if err != nil {
		return err
	}
	newPass, err := a.readRequired("New password")
	if err != nil {
		return err
	}
	return a.servicePorts.UserProfile.ChangePasswordUser(ctx, claims.UserID, oldPass, newPass)
}

func (a *App) deleteProfile(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	confirmed, err := a.readBool("Delete current user? [y/N]")
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}
	if err := a.servicePorts.UserProfile.DeleteUserByID(ctx, claims.UserID); err != nil {
		return err
	}
	a.claims = nil
	a.token = ""
	return nil
}

func (a *App) productsMenu(ctx context.Context) error {
	a.println("")
	a.println("1. List products")
	a.println("2. Show product")
	a.println("3. Product price history")
	a.println("4. Product reviews")
	a.println("5. Create product")
	a.println("6. Update product")
	a.println("7. Delete product")
	a.println("0. Back")
	choice, err := a.readLine("Select")
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		return a.listProducts(ctx)
	case "2":
		return a.showProduct(ctx)
	case "3":
		return a.productPriceHistory(ctx)
	case "4":
		return a.productReviews(ctx)
	case "5":
		return a.createProduct(ctx)
	case "6":
		return a.updateProduct(ctx)
	case "7":
		return a.deleteProduct(ctx)
	case "0":
		return nil
	default:
		a.println("Unknown command")
		return nil
	}
}

func (a *App) listProducts(ctx context.Context) error {
	name, err := a.readOptionalString("Name contains")
	if err != nil {
		return err
	}
	minPrice, err := a.readOptionalDecimal("Min price")
	if err != nil {
		return err
	}
	maxPrice, err := a.readOptionalDecimal("Max price")
	if err != nil {
		return err
	}
	sellerID, err := a.readOptionalInt64("Seller ID")
	if err != nil {
		return err
	}
	categoryLine, err := a.readLine("Categories, comma-separated")
	if err != nil {
		return err
	}
	var categories []string
	for _, category := range strings.Split(categoryLine, ",") {
		category = strings.TrimSpace(category)
		if category != "" {
			categories = append(categories, category)
		}
	}
	pg := m.PaginationOpts{Page: 1, Limit: 20}
	products, err := a.servicePorts.ProductCatalog.GetProducts(ctx, m.CatalogOptions{
		Categories: categories,
		MinPrice:   minPrice,
		MaxPrice:   maxPrice,
		FilterName: name,
		SellerID:   sellerID,
		Pagination: &pg,
	})
	if err != nil {
		return err
	}
	for _, product := range products {
		a.printProduct(product)
	}
	return nil
}

func (a *App) showProduct(ctx context.Context) error {
	id, err := a.readInt64("Product ID")
	if err != nil {
		return err
	}
	product, err := a.servicePorts.ProductDetails.GetProductByID(ctx, id)
	if err != nil {
		return err
	}
	a.printProduct(product)
	return nil
}

func (a *App) productPriceHistory(ctx context.Context) error {
	id, err := a.readInt64("Product ID")
	if err != nil {
		return err
	}
	dateFrom, err := a.readDate("Date from [YYYY-MM-DD]")
	if err != nil {
		return err
	}
	dateTo, err := a.readDate("Date to [YYYY-MM-DD]")
	if err != nil {
		return err
	}
	history, err := a.servicePorts.ProductDetails.GetProductPriceHistory(ctx, id, dateFrom, dateTo)
	if err != nil {
		return err
	}
	for _, item := range history {
		a.printf("#%d product=%d %s -> %s at %s by %s\n",
			item.ID, item.ProductID, item.OldPrice, item.NewPrice, item.ChangedAt.Format(time.RFC3339), item.ChangedBy)
	}
	return nil
}

func (a *App) productReviews(ctx context.Context) error {
	id, err := a.readInt64("Product ID")
	if err != nil {
		return err
	}
	reviews, err := a.servicePorts.ProductDetails.GetReviewsByProductID(ctx, id, m.PaginationOpts{Page: 1, Limit: 20})
	if err != nil {
		return err
	}
	for _, review := range reviews {
		a.printReview(review)
	}
	return nil
}

func (a *App) createProduct(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	sellerID, err := a.readInt64("Seller ID")
	if err != nil {
		return err
	}
	name, err := a.readRequired("Name")
	if err != nil {
		return err
	}
	description, err := a.readOptionalString("Description")
	if err != nil {
		return err
	}
	price, err := a.readDecimal("Price")
	if err != nil {
		return err
	}
	stock, err := a.readInt("Stock quantity")
	if err != nil {
		return err
	}
	id, err := a.servicePorts.ProductManagement.CreateProduct(ctx, claims.UserID, m.ProductCreate{
		SellerID:      sellerID,
		Name:          name,
		Description:   description,
		Price:         price,
		StockQuantity: stock,
	})
	if err != nil {
		return err
	}
	a.printf("Created product id=%d\n", id)
	return nil
}

func (a *App) updateProduct(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	id, err := a.readInt64("Product ID")
	if err != nil {
		return err
	}
	name, err := a.readOptionalString("New name")
	if err != nil {
		return err
	}
	description, err := a.readOptionalString("New description")
	if err != nil {
		return err
	}
	price, err := a.readOptionalDecimal("New price")
	if err != nil {
		return err
	}
	stock, err := a.readOptionalInt("New stock quantity")
	if err != nil {
		return err
	}
	changedBy := fmt.Sprintf("%s:%d:techui", claims.Role, claims.UserID)
	product, err := a.servicePorts.ProductManagement.UpdateProduct(ctx, claims.UserID, id, m.ProductUpdate{
		Name:          name,
		Description:   description,
		Price:         price,
		StockQuantity: stock,
		ChangedBy:     &changedBy,
	})
	if err != nil {
		return err
	}
	a.printProduct(product)
	return nil
}

func (a *App) deleteProduct(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	id, err := a.readInt64("Product ID")
	if err != nil {
		return err
	}
	return a.servicePorts.ProductManagement.DeleteProductByID(ctx, claims.UserID, id)
}

func (a *App) categoriesMenu(ctx context.Context) error {
	a.println("")
	a.println("1. List categories")
	a.println("2. Show category")
	a.println("3. Create category")
	a.println("4. Update category")
	a.println("5. Delete category")
	a.println("0. Back")
	choice, err := a.readLine("Select")
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		return a.listCategories(ctx)
	case "2":
		return a.showCategory(ctx)
	case "3":
		return a.createCategory(ctx)
	case "4":
		return a.updateCategory(ctx)
	case "5":
		return a.deleteCategory(ctx)
	case "0":
		return nil
	default:
		a.println("Unknown command")
		return nil
	}
}

func (a *App) listCategories(ctx context.Context) error {
	categories, err := a.servicePorts.CategoryBrowser.GetCategories(ctx, m.PaginationOpts{Page: 1, Limit: 100})
	if err != nil {
		return err
	}
	for _, category := range categories {
		a.printCategory(category)
	}
	return nil
}

func (a *App) showCategory(ctx context.Context) error {
	id, err := a.readInt64("Category ID")
	if err != nil {
		return err
	}
	category, err := a.servicePorts.CategoryBrowser.GetCategoryByID(ctx, id)
	if err != nil {
		return err
	}
	a.printCategory(category)
	return nil
}

func (a *App) createCategory(ctx context.Context) error {
	if !a.requireRole(m.RoleAdmin) {
		return nil
	}
	parentID, err := a.readOptionalInt64("Parent ID")
	if err != nil {
		return err
	}
	name, err := a.readRequired("Name")
	if err != nil {
		return err
	}
	description, err := a.readOptionalString("Description")
	if err != nil {
		return err
	}
	id, err := a.servicePorts.CategoryManagement.CreateCategory(ctx, m.CategoryCreate{
		ParentID:    parentID,
		Name:        name,
		Description: description,
	})
	if err != nil {
		return err
	}
	a.printf("Created category id=%d\n", id)
	return nil
}

func (a *App) updateCategory(ctx context.Context) error {
	if !a.requireRole(m.RoleAdmin) {
		return nil
	}
	id, err := a.readInt64("Category ID")
	if err != nil {
		return err
	}
	parentID, err := a.readOptionalInt64("New parent ID")
	if err != nil {
		return err
	}
	name, err := a.readOptionalString("New name")
	if err != nil {
		return err
	}
	description, err := a.readOptionalString("New description")
	if err != nil {
		return err
	}
	category, err := a.servicePorts.CategoryManagement.UpdateCategory(ctx, id, m.CategoryUpdate{
		ParentID:    parentID,
		Name:        name,
		Description: description,
	})
	if err != nil {
		return err
	}
	a.printCategory(category)
	return nil
}

func (a *App) deleteCategory(ctx context.Context) error {
	if !a.requireRole(m.RoleAdmin) {
		return nil
	}
	id, err := a.readInt64("Category ID")
	if err != nil {
		return err
	}
	return a.servicePorts.CategoryManagement.DeleteCategoryByID(ctx, id)
}

func (a *App) addressesMenu(ctx context.Context) error {
	a.println("")
	a.println("1. List addresses")
	a.println("2. Create address")
	a.println("3. Update address")
	a.println("4. Delete address")
	a.println("0. Back")
	choice, err := a.readLine("Select")
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		return a.listAddresses(ctx)
	case "2":
		return a.createAddress(ctx)
	case "3":
		return a.updateAddress(ctx)
	case "4":
		return a.deleteAddress(ctx)
	case "0":
		return nil
	default:
		a.println("Unknown command")
		return nil
	}
}

func (a *App) listAddresses(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	addresses, err := a.servicePorts.AddressBook.GetAddressesByUserID(ctx, claims.UserID)
	if err != nil {
		return err
	}
	for _, address := range addresses {
		a.printAddress(address)
	}
	return nil
}

func (a *App) createAddress(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	city, err := a.readRequired("City")
	if err != nil {
		return err
	}
	street, err := a.readRequired("Street")
	if err != nil {
		return err
	}
	zip, err := a.readRequired("Zip code")
	if err != nil {
		return err
	}
	isDefault, err := a.readBool("Default address? [y/N]")
	if err != nil {
		return err
	}
	id, err := a.servicePorts.AddressBook.CreateAddress(ctx, m.AddressCreate{
		UserID:    claims.UserID,
		City:      city,
		Street:    street,
		ZipCode:   zip,
		IsDefault: isDefault,
	})
	if err != nil {
		return err
	}
	a.printf("Created address id=%d\n", id)
	return nil
}

func (a *App) updateAddress(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	id, err := a.readInt64("Address ID")
	if err != nil {
		return err
	}
	city, err := a.readOptionalString("New city")
	if err != nil {
		return err
	}
	street, err := a.readOptionalString("New street")
	if err != nil {
		return err
	}
	zip, err := a.readOptionalString("New zip code")
	if err != nil {
		return err
	}
	isDefault, err := a.readOptionalBool("New default flag [y/n]")
	if err != nil {
		return err
	}
	address, err := a.servicePorts.AddressBook.UpdateAddress(ctx, claims.UserID, id, m.AddressUpdate{
		City:      city,
		Street:    street,
		ZipCode:   zip,
		IsDefault: isDefault,
	})
	if err != nil {
		return err
	}
	a.printAddress(address)
	return nil
}

func (a *App) deleteAddress(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	id, err := a.readInt64("Address ID")
	if err != nil {
		return err
	}
	return a.servicePorts.AddressBook.DeleteAddressByID(ctx, claims.UserID, id)
}

func (a *App) ordersMenu(ctx context.Context) error {
	a.println("")
	a.println("1. Show cart")
	a.println("2. Add item to cart")
	a.println("3. Change cart item quantity")
	a.println("4. Delete cart item")
	a.println("5. Checkout")
	a.println("6. List my orders")
	a.println("7. Show order")
	a.println("8. Show order items")
	a.println("9. Get payment URL")
	a.println("10. Process mock-bank payment token")
	a.println("11. Pay order directly")
	a.println("12. Cancel order")
	a.println("0. Back")
	choice, err := a.readLine("Select")
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		return a.showCart(ctx)
	case "2":
		return a.addCartItem(ctx)
	case "3":
		return a.changeCartItem(ctx)
	case "4":
		return a.deleteCartItem(ctx)
	case "5":
		return a.checkout(ctx)
	case "6":
		return a.listOrders(ctx)
	case "7":
		return a.showOrder(ctx)
	case "8":
		return a.showOrderItems(ctx)
	case "9":
		return a.paymentURL(ctx)
	case "10":
		return a.processPayment(ctx)
	case "11":
		return a.payOrderDirectly(ctx)
	case "12":
		return a.cancelOrder(ctx)
	case "0":
		return nil
	default:
		a.println("Unknown command")
		return nil
	}
}

func (a *App) showCart(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	cart, err := a.servicePorts.Cart.GetCart(ctx, claims.UserID)
	if err != nil {
		return err
	}
	a.printOrder(cart)
	items, err := a.servicePorts.Cart.GetOrderItemsByOrderID(ctx, cart.ID)
	if err != nil {
		return err
	}
	for _, item := range items {
		a.printOrderItem(item)
	}
	return nil
}

func (a *App) addCartItem(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	productID, err := a.readInt64("Product ID")
	if err != nil {
		return err
	}
	quantity, err := a.readInt("Quantity")
	if err != nil {
		return err
	}
	return a.servicePorts.Cart.AddItemToCart(ctx, claims.UserID, productID, quantity)
}

func (a *App) changeCartItem(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	itemID, err := a.readInt64("Order item ID")
	if err != nil {
		return err
	}
	quantity, err := a.readInt("New quantity")
	if err != nil {
		return err
	}
	return a.servicePorts.Cart.ChangeQuantityCartItem(ctx, claims.UserID, itemID, quantity)
}

func (a *App) deleteCartItem(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	itemID, err := a.readInt64("Order item ID")
	if err != nil {
		return err
	}
	return a.servicePorts.Cart.DeleteCartItem(ctx, claims.UserID, itemID)
}

func (a *App) checkout(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	addressID, err := a.readInt64("Address ID")
	if err != nil {
		return err
	}
	orderIDs, err := a.servicePorts.Cart.Checkout(ctx, claims.UserID, addressID)
	if err != nil {
		return err
	}
	a.printf("Created pending orders: %v\n", orderIDs)
	return nil
}

func (a *App) listOrders(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	orders, err := a.servicePorts.BuyerOrders.GetOrdersByUserID(ctx, claims.UserID, m.PaginationOpts{Page: 1, Limit: 20})
	if err != nil {
		return err
	}
	for _, order := range orders {
		a.printOrder(order)
	}
	return nil
}

func (a *App) showOrder(ctx context.Context) error {
	id, err := a.readInt64("Order ID")
	if err != nil {
		return err
	}
	order, err := a.servicePorts.BuyerOrders.GetOrderByID(ctx, id)
	if err != nil {
		return err
	}
	a.printOrder(order)
	return nil
}

func (a *App) showOrderItems(ctx context.Context) error {
	id, err := a.readInt64("Order ID")
	if err != nil {
		return err
	}
	items, err := a.servicePorts.BuyerOrders.GetOrderItemsByOrderID(ctx, id)
	if err != nil {
		return err
	}
	for _, item := range items {
		a.printOrderItem(item)
	}
	return nil
}

func (a *App) paymentURL(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	orderID, err := a.readInt64("Order ID")
	if err != nil {
		return err
	}
	paymentURL, expiresAt, err := a.servicePorts.Payments.GetOrderPaymentURL(ctx, orderID, claims.UserID)
	if err != nil {
		return err
	}
	a.printf("Payment URL: %s\nExpires at: %s\n", paymentURL, expiresAt.Format(time.RFC3339))
	processNow, err := a.readBool("Process this mock payment now? [y/N]")
	if err != nil {
		return err
	}
	if !processNow {
		return nil
	}
	token, err := tokenFromPaymentURL(paymentURL)
	if err != nil {
		return err
	}
	return a.servicePorts.Payments.ProcessOrderPayment(ctx, token)
}

func (a *App) processPayment(ctx context.Context) error {
	token, err := a.readRequired("Mock-bank token or payment URL")
	if err != nil {
		return err
	}
	if strings.Contains(token, "://") {
		token, err = tokenFromPaymentURL(token)
		if err != nil {
			return err
		}
	}
	return a.servicePorts.Payments.ProcessOrderPayment(ctx, token)
}

func (a *App) payOrderDirectly(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	orderID, err := a.readInt64("Order ID")
	if err != nil {
		return err
	}
	return a.servicePorts.BuyerOrders.PayOrder(ctx, orderID, claims.UserID)
}

func (a *App) cancelOrder(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	orderID, err := a.readInt64("Order ID")
	if err != nil {
		return err
	}
	return a.servicePorts.BuyerOrders.CancelOrder(ctx, orderID, claims.UserID)
}

func (a *App) sellerMenu(ctx context.Context) error {
	a.println("")
	a.println("1. My seller profile")
	a.println("2. Create seller profile")
	a.println("3. Show seller by ID")
	a.println("4. Update seller")
	a.println("5. Delete seller")
	a.println("6. Seller stats")
	a.println("7. Seller orders")
	a.println("8. Ship order")
	a.println("9. Deliver order")
	a.println("0. Back")
	choice, err := a.readLine("Select")
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		return a.mySeller(ctx)
	case "2":
		return a.createSeller(ctx)
	case "3":
		return a.showSeller(ctx)
	case "4":
		return a.updateSeller(ctx)
	case "5":
		return a.deleteSeller(ctx)
	case "6":
		return a.sellerStats(ctx)
	case "7":
		return a.sellerOrders(ctx)
	case "8":
		return a.shipOrder(ctx)
	case "9":
		return a.deliverOrder(ctx)
	case "0":
		return nil
	default:
		a.println("Unknown command")
		return nil
	}
}

func (a *App) mySeller(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	seller, err := a.servicePorts.SellerProfile.GetSellerByUserID(ctx, claims.UserID)
	if err != nil {
		return err
	}
	a.printSeller(seller)
	return nil
}

func (a *App) createSeller(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	company, err := a.readRequired("Company name")
	if err != nil {
		return err
	}
	description, err := a.readOptionalString("Description")
	if err != nil {
		return err
	}
	id, err := a.servicePorts.SellerProfile.CreateSeller(ctx, m.SellerCreate{
		UserID:      claims.UserID,
		CompanyName: company,
		Description: description,
	})
	if err != nil {
		return err
	}
	a.printf("Created seller id=%d\n", id)
	return nil
}

func (a *App) showSeller(ctx context.Context) error {
	id, err := a.readInt64("Seller ID")
	if err != nil {
		return err
	}
	seller, err := a.servicePorts.SellerProfile.GetSellerByID(ctx, id)
	if err != nil {
		return err
	}
	a.printSeller(seller)
	return nil
}

func (a *App) updateSeller(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	id, err := a.readInt64("Seller ID")
	if err != nil {
		return err
	}
	company, err := a.readOptionalString("New company name")
	if err != nil {
		return err
	}
	description, err := a.readOptionalString("New description")
	if err != nil {
		return err
	}
	seller, err := a.servicePorts.SellerProfile.UpdateSeller(ctx, claims.UserID, id, m.SellerUpdate{
		CompanyName: company,
		Description: description,
	})
	if err != nil {
		return err
	}
	a.printSeller(seller)
	return nil
}

func (a *App) deleteSeller(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	id, err := a.readInt64("Seller ID")
	if err != nil {
		return err
	}
	return a.servicePorts.SellerProfile.DeleteSellerByID(ctx, claims.UserID, id)
}

func (a *App) sellerStats(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	id, err := a.readInt64("Seller ID")
	if err != nil {
		return err
	}
	dateFrom, err := a.readDate("Date from [YYYY-MM-DD]")
	if err != nil {
		return err
	}
	dateTo, err := a.readDate("Date to [YYYY-MM-DD]")
	if err != nil {
		return err
	}
	stats, err := a.servicePorts.SellerStatistics.GetSellerStats(ctx, claims.UserID, id, dateFrom, dateTo)
	if err != nil {
		return err
	}
	a.printf("orders=%d revenue=%s avg=%s top_product=%s\n",
		stats.TotalOrders, stats.TotalRevenue, stats.AvgOrderValue, stats.TopProductName)
	return nil
}

func (a *App) sellerOrders(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	orders, err := a.servicePorts.SellerOrders.GetSellerOrdersByUserID(ctx, claims.UserID, m.PaginationOpts{Page: 1, Limit: 20})
	if err != nil {
		return err
	}
	for _, order := range orders {
		a.printOrder(order)
	}
	return nil
}

func (a *App) shipOrder(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	id, err := a.readInt64("Order ID")
	if err != nil {
		return err
	}
	return a.servicePorts.SellerOrders.ShipOrder(ctx, id, claims.UserID)
}

func (a *App) deliverOrder(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	id, err := a.readInt64("Order ID")
	if err != nil {
		return err
	}
	return a.servicePorts.SellerOrders.DeliverOrder(ctx, id, claims.UserID)
}

func (a *App) reviewsMenu(ctx context.Context) error {
	a.println("")
	a.println("1. Create review")
	a.println("2. Show review")
	a.println("3. Update review")
	a.println("4. Delete review")
	a.println("0. Back")
	choice, err := a.readLine("Select")
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		return a.createReview(ctx)
	case "2":
		return a.showReview(ctx)
	case "3":
		return a.updateReview(ctx)
	case "4":
		return a.deleteReview(ctx)
	case "0":
		return nil
	default:
		a.println("Unknown command")
		return nil
	}
}

func (a *App) createReview(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	productID, err := a.readInt64("Product ID")
	if err != nil {
		return err
	}
	rating, err := a.readInt("Rating [1..5]")
	if err != nil {
		return err
	}
	comment, err := a.readOptionalString("Comment")
	if err != nil {
		return err
	}
	id, err := a.servicePorts.ReviewManagement.CreateReview(ctx, m.ReviewCreate{
		UserID:    claims.UserID,
		ProductID: productID,
		Rating:    int8(rating),
		Comment:   comment,
	})
	if err != nil {
		return err
	}
	a.printf("Created review id=%d\n", id)
	return nil
}

func (a *App) showReview(ctx context.Context) error {
	id, err := a.readInt64("Review ID")
	if err != nil {
		return err
	}
	review, err := a.servicePorts.ReviewManagement.GetReviewByID(ctx, id)
	if err != nil {
		return err
	}
	a.printReview(review)
	return nil
}

func (a *App) updateReview(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	id, err := a.readInt64("Review ID")
	if err != nil {
		return err
	}
	rating, err := a.readOptionalInt("New rating [1..5]")
	if err != nil {
		return err
	}
	var rating8 *int8
	if rating != nil {
		value := int8(*rating)
		rating8 = &value
	}
	comment, err := a.readOptionalString("New comment")
	if err != nil {
		return err
	}
	review, err := a.servicePorts.ReviewManagement.UpdateReview(ctx, claims.UserID, id, m.ReviewUpdate{
		Rating:  rating8,
		Comment: comment,
	})
	if err != nil {
		return err
	}
	a.printReview(review)
	return nil
}

func (a *App) deleteReview(ctx context.Context) error {
	claims, ok := a.requireAuth()
	if !ok {
		return nil
	}
	id, err := a.readInt64("Review ID")
	if err != nil {
		return err
	}
	return a.servicePorts.ReviewManagement.DeleteReviewByID(ctx, claims.UserID, id)
}

func (a *App) adminMenu(ctx context.Context) error {
	if !a.requireRole(m.RoleAdmin) {
		return nil
	}
	a.println("")
	a.println("1. List users")
	a.println("2. Show user")
	a.println("3. Update user role")
	a.println("4. Delete user")
	a.println("0. Back")
	choice, err := a.readLine("Select")
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		return a.adminListUsers(ctx)
	case "2":
		return a.adminShowUser(ctx)
	case "3":
		return a.adminUpdateUserRole(ctx)
	case "4":
		return a.adminDeleteUser(ctx)
	case "0":
		return nil
	default:
		a.println("Unknown command")
		return nil
	}
}

func (a *App) adminListUsers(ctx context.Context) error {
	roleLine, err := a.readLine("Role filter [buyer/seller/admin/analyst]")
	if err != nil {
		return err
	}
	var role *string
	if roleLine != "" {
		role = &roleLine
	}
	search, err := a.readOptionalString("Search")
	if err != nil {
		return err
	}
	users, err := a.servicePorts.UserAdministration.GetUsers(ctx, m.UserListOptions{
		Search:     search,
		Role:       role,
		Pagination: m.PaginationOpts{Page: 1, Limit: 50},
	})
	if err != nil {
		return err
	}
	for _, user := range users {
		a.printUser(user)
	}
	return nil
}

func (a *App) adminShowUser(ctx context.Context) error {
	id, err := a.readInt64("User ID")
	if err != nil {
		return err
	}
	user, err := a.servicePorts.UserAdministration.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	a.printUser(user)
	return nil
}

func (a *App) adminUpdateUserRole(ctx context.Context) error {
	id, err := a.readInt64("User ID")
	if err != nil {
		return err
	}
	role, err := a.readRole("New role [buyer/seller/admin/analyst]", m.RoleBuyer)
	if err != nil {
		return err
	}
	user, err := a.servicePorts.UserAdministration.UpdateUser(ctx, id, m.UserUpdate{Role: &role})
	if err != nil {
		return err
	}
	a.printUser(user)
	return nil
}

func (a *App) adminDeleteUser(ctx context.Context) error {
	id, err := a.readInt64("User ID")
	if err != nil {
		return err
	}
	return a.servicePorts.UserAdministration.DeleteUserByID(ctx, id)
}

func (a *App) run(ctx context.Context, fn func(context.Context) error) {
	err := fn(ctx)
	switch {
	case err == nil:
		a.println("OK")
	case errors.Is(err, io.EOF):
		return
	default:
		a.printf("ERROR: %v\n", err)
	}
}

func (a *App) requireAuth() (m.TokenClaims, bool) {
	if a.claims == nil {
		a.println("Authentication required")
		return m.TokenClaims{}, false
	}
	return *a.claims, true
}

func (a *App) requireRole(role m.UserRole) bool {
	claims, ok := a.requireAuth()
	if !ok {
		return false
	}
	if claims.Role != role {
		a.printf("Role %s required\n", role)
		return false
	}
	return true
}

func (a *App) readLine(label string) (string, error) {
	if label != "" {
		a.printf("%s: ", label)
	}
	line, err := a.in.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && line != "" {
			return strings.TrimSpace(line), nil
		}
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (a *App) readRequired(label string) (string, error) {
	for {
		value, err := a.readLine(label)
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
		a.println("Value is required")
	}
}

func (a *App) readOptionalString(label string) (*string, error) {
	value, err := a.readLine(label)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	return &value, nil
}

func (a *App) readInt64(label string) (int64, error) {
	for {
		value, err := a.readRequired(label)
		if err != nil {
			return 0, err
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return parsed, nil
		}
		a.println("Enter integer value")
	}
}

func (a *App) readOptionalInt64(label string) (*int64, error) {
	value, err := a.readLine(label)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (a *App) readInt(label string) (int, error) {
	for {
		value, err := a.readRequired(label)
		if err != nil {
			return 0, err
		}
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed, nil
		}
		a.println("Enter integer value")
	}
}

func (a *App) readOptionalInt(label string) (*int, error) {
	value, err := a.readLine(label)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (a *App) readDecimal(label string) (decimal.Decimal, error) {
	for {
		value, err := a.readRequired(label)
		if err != nil {
			return decimal.Zero, err
		}
		parsed, err := decimal.NewFromString(value)
		if err == nil {
			return parsed, nil
		}
		a.println("Enter decimal value")
	}
}

func (a *App) readOptionalDecimal(label string) (*decimal.Decimal, error) {
	value, err := a.readLine(label)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (a *App) readBool(label string) (bool, error) {
	value, err := a.readLine(label)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(value) {
	case "y", "yes", "true", "1", "да":
		return true, nil
	default:
		return false, nil
	}
}

func (a *App) readOptionalBool(label string) (*bool, error) {
	value, err := a.readLine(label)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	result := false
	switch strings.ToLower(value) {
	case "y", "yes", "true", "1", "да":
		result = true
	case "n", "no", "false", "0", "нет":
		result = false
	default:
		return nil, fmt.Errorf("invalid boolean value: %s", value)
	}
	return &result, nil
}

func (a *App) readDate(label string) (time.Time, error) {
	for {
		value, err := a.readRequired(label)
		if err != nil {
			return time.Time{}, err
		}
		parsed, err := time.Parse(time.DateOnly, value)
		if err == nil {
			return parsed, nil
		}
		a.println("Enter date as YYYY-MM-DD")
	}
}

func (a *App) readRole(label string, defaultRole m.UserRole) (m.UserRole, error) {
	value, err := a.readLine(label)
	if err != nil {
		return "", err
	}
	if value == "" {
		return defaultRole, nil
	}
	role := m.UserRole(value)
	switch role {
	case m.RoleAdmin, m.RoleAnalyst, m.RoleBuyer, m.RoleSeller:
		return role, nil
	default:
		return "", fmt.Errorf("unknown role: %s", value)
	}
}

func (a *App) printUser(user m.User) {
	a.printf("#%d %s <%s> role=%s phone=%s created=%s\n",
		user.ID, user.FullName, user.Email, user.Role, stringPtr(user.Phone), user.CreatedAt.Format(time.DateOnly))
}

func (a *App) printSeller(seller m.Seller) {
	a.printf("#%d user=%d company=%s rating=%s description=%s\n",
		seller.ID, seller.UserID, seller.CompanyName, floatPtr(seller.Rating), stringPtr(seller.Description))
}

func (a *App) printProduct(product m.Product) {
	a.printf("#%d seller=%d name=%s price=%s stock=%d rating=%s description=%s\n",
		product.ID, product.SellerID, product.Name, product.Price, product.StockQuantity, floatPtr(product.Rating), stringPtr(product.Description))
}

func (a *App) printCategory(category m.Category) {
	a.printf("#%d parent=%s name=%s description=%s\n",
		category.ID, int64Ptr(category.ParentID), category.Name, stringPtr(category.Description))
}

func (a *App) printAddress(address m.Address) {
	a.printf("#%d user=%d %s, %s, %s default=%t\n",
		address.ID, address.UserID, address.City, address.Street, address.ZipCode, address.IsDefault)
}

func (a *App) printOrder(order m.Order) {
	a.printf("#%d user=%d seller=%s address=%s status=%s total=%s created=%s\n",
		order.ID, order.UserID, int64Ptr(order.SellerID), int64Ptr(order.AddressID), order.Status, order.TotalAmount, order.CreatedAt.Format(time.RFC3339))
}

func (a *App) printOrderItem(item m.OrderItem) {
	a.printf("#%d order=%d product=%d quantity=%d price_at_purchase=%s\n",
		item.ID, item.OrderID, item.ProductID, item.Quantity, item.PriceAtPurchase)
}

func (a *App) printReview(review m.Review) {
	a.printf("#%d user=%d product=%d rating=%d comment=%s created=%s\n",
		review.ID, review.UserID, review.ProductID, review.Rating, stringPtr(review.Comment), review.CreatedAt.Format(time.DateOnly))
}

func (a *App) println(args ...any) {
	fmt.Fprintln(a.out, args...)
}

func (a *App) printf(format string, args ...any) {
	fmt.Fprintf(a.out, format, args...)
}

func stringPtr(value *string) string {
	if value == nil {
		return "-"
	}
	return *value
}

func int64Ptr(value *int64) string {
	if value == nil {
		return "-"
	}
	return strconv.FormatInt(*value, 10)
}

func floatPtr(value *float32) string {
	if value == nil {
		return "-"
	}
	return strconv.FormatFloat(float64(*value), 'f', 2, 32)
}

func tokenFromPaymentURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	token := parsed.Query().Get("token")
	if token == "" {
		return "", errors.New("payment URL has no token query parameter")
	}
	return token, nil
}
