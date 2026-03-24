package service

import (
	"errors"
)

// service layer errors
var (
	// auth
	ErrSign               = errors.New("Failed to sign")
	ErrLogin              = errors.New("Login failed")
	ErrGenerateToken      = errors.New("Failed to generate token")
	ErrParseToken         = errors.New("Failed to parse token")
	ErrWrongSigningMethod = errors.New("Wrong signing method algorithm")
	ErrRegistration       = errors.New("Failed to register user")
	ErrLogout             = errors.New("Failed to logout")
	ErrTokenBlocked       = errors.New("Token has been revoked")

	// users
	ErrAccountWithEmailAlreadyExists = errors.New("Account with this email already exists")
	ErrHashingPassword        = errors.New("Error occured when hashing passowrd")
	ErrCreateUser             = errors.New("Error occured when creating user")
	ErrUpdateUser             = errors.New("Error occured when updating user")
	ErrGetUserByID            = errors.New("Error occured when getting user by id")
	ErrGetUserByEmail         = errors.New("Error occured when getting user by email")
	ErrDeleteUser             = errors.New("Error occured when deleting user")
	ErrChangePasswordUser     = errors.New("Error occured when changing user password")
	ErrUserNotFound           = errors.New("User not found")
	ErrWrongPassword          = errors.New("Wrong password")

	// addresses
	ErrGetAddressByID       = errors.New("Failed to get address by id")
	ErrGetAddressesByUserID = errors.New("Failed to get addresses by user id")
	ErrCreateAddress        = errors.New("Failed to create address")
	ErrUpdateAddress        = errors.New("Failed to update address")
	ErrDeleteAddressByID    = errors.New("Failed to delete address by id")
	ErrNotYourAddress       = errors.New("It's not your address")
	ErrAddressNotFound      = errors.New("Address not found")

	// sellers
	ErrGetSellerByID    = errors.New("Failed to get seller by id")
	ErrGetSellerStats   = errors.New("Failed to get seller stats")
	ErrCreateSeller     = errors.New("Failed to create seller")
	ErrUpdateSeller     = errors.New("Failed to update seller")
	ErrDeleteSellerByID = errors.New("Failed to delete seller by id")
	ErrNotYourSeller    = errors.New("It's not your seller")
	ErrSellerNotFound   = errors.New("Seller not found")

	// orders
	ErrGetOrderByID           = errors.New("Failed to get order by id")
	ErrGetOrderItemByID       = errors.New("Failed to get order item by id")
	ErrGetOrdersByUserID      = errors.New("Failed to get orders by user id")
	ErrGetOrderItemsByOrderID = errors.New("Failed to get order items by order id")
	ErrGetCart                = errors.New("Failed to get cart order")
	ErrCreateOrder            = errors.New("Failed to create order")
	ErrCreateOrderItem        = errors.New("Failed to create order item")
	ErrUpdateOrder            = errors.New("Failed to update order")
	ErrUpdateOrderItem        = errors.New("Failed to update order item")
	ErrDeleteOrderByID        = errors.New("Failed to delete order by id")
	ErrDeleteOrderItemByID    = errors.New("Failed to delete order item by id")
	ErrNoOrders               = errors.New("User does not have orders")
	ErrOrderNotFound          = errors.New("Order not found")
	ErrOrderItemNotFound      = errors.New("Order item not found")
	ErrCartNotFound           = errors.New("Cart not found")
	ErrEmptyCart              = errors.New("Cart is empty")
	ErrNotYourOrder           = errors.New("It's not your order")
	ErrSellerNotSet           = errors.New("Seller is not set for this order")
	ErrOrderStatusInvalid     = errors.New("Invalid order status")

	//categories
	ErrGetCategories    = errors.New("Failed to get categories")
	ErrGetCategoryByID  = errors.New("Failed to get category by id")
	ErrCreateCategory   = errors.New("Failed to create category")
	ErrUpdateCategory   = errors.New("Failed to update category")
	ErrDeleteCategory   = errors.New("Failed to delete category")
	ErrCategoryNotFound = errors.New("Category not found")

	// products
	ErrGetProducts            = errors.New("Failed to get products")
	ErrGetProductByID         = errors.New("Failed to get product by id")
	ErrGetProductPriceHistory = errors.New("Failed to get product price history")
	ErrCreateProduct          = errors.New("Failed to create product")
	ErrUpdateProduct          = errors.New("Failed to update product")
	ErrDeleteProduct          = errors.New("Failed to delete product")
	ErrQuantityTooBig         = errors.New("Quantity too big, stock is less than that")
	ErrProductNotFound        = errors.New("Product not found")

	// review
	ErrCreateReview          = errors.New("Failed to create review")
	ErrGetReviewByID         = errors.New("Failed to get review by id")
	ErrUpdateReview          = errors.New("Failed to update review")
	ErrDeleteReviewByID      = errors.New("Failed to delete review")
	ErrGetReviewsByProductID = errors.New("Failed to get reviews by product id")
	ErrNotYourReview         = errors.New("It's not your review")
	ErrReviewNotFound        = errors.New("Review not found")
)

// For repos usage
var (
	ErrNotFound          = errors.New("Not found")
	ErrNoChangesInUpdate = errors.New("All update fields are nil")
)
