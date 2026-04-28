package service

import (
	"errors"
)

// service layer errors
var (
	// authorization
	ErrPermissionDenied = errors.New("Permission denied")

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
	ErrHashingPassword               = errors.New("Error occured when hashing passowrd")
	ErrCreateUser                    = errors.New("Error occured when creating user")
	ErrUpdateUser                    = errors.New("Error occured when updating user")
	ErrGetUserByID                   = errors.New("Error occured when getting user by id")
	ErrGetUserByEmail                = errors.New("Error occured when getting user by email")
	ErrGetUsers                      = errors.New("Error occured when getting users")
	ErrDeleteUser                    = errors.New("Error occured when deleting user")
	ErrChangePasswordUser            = errors.New("Error occured when changing user password")
	ErrUserNotFound                  = errors.New("User not found")
	ErrWrongPassword                 = errors.New("Wrong password")
	ErrPhoneAlreadyExists            = errors.New("This phone number is already in use")
	ErrEmailAlreadyInUse             = errors.New("This email is already in use")
	ErrNotYourUser                   = errors.New("It's not your user")

	// addresses
	ErrGetAddressByID       = errors.New("Failed to get address by id")
	ErrGetAddressesByUserID = errors.New("Failed to get addresses by user id")
	ErrCreateAddress        = errors.New("Failed to create address")
	ErrUpdateAddress        = errors.New("Failed to update address")
	ErrDeleteAddressByID    = errors.New("Failed to delete address by id")
	ErrNotYourAddress       = errors.New("It's not your address")
	ErrAddressNotFound      = errors.New("Address not found")

	// sellers
	ErrGetSellerByID     = errors.New("Failed to get seller by id")
	ErrGetSellerByUserID = errors.New("Failed to get seller by user id")
	ErrGetSellerStats    = errors.New("Failed to get seller stats")
	ErrCreateSeller      = errors.New("Failed to create seller")
	ErrUpdateSeller      = errors.New("Failed to update seller")
	ErrDeleteSellerByID  = errors.New("Failed to delete seller by id")
	ErrNotYourSeller     = errors.New("It's not your seller")
	ErrSellerNotFound    = errors.New("Seller not found")

	// orders
	ErrGetOrderByID           = errors.New("Failed to get order by id")
	ErrGetOrderItemByID       = errors.New("Failed to get order item by id")
	ErrGetAdminOrders         = errors.New("Failed to get admin orders")
	ErrGetOrdersByUserID      = errors.New("Failed to get orders by user id")
	ErrGetOrdersBySellerID    = errors.New("Failed to get orders by seller id")
	ErrGetOrderItemsByOrderID = errors.New("Failed to get order items by order id")
	ErrGetCart                = errors.New("Failed to get cart order")
	ErrCreateOrder            = errors.New("Failed to create order")
	ErrCreateOrderItem        = errors.New("Failed to create order item")
	ErrUpdateOrder            = errors.New("Failed to update order")
	ErrUpdateOrderItem        = errors.New("Failed to update order item")
	ErrDeleteOrderByID        = errors.New("Failed to delete order by id")
	ErrDeleteOrderItemByID    = errors.New("Failed to delete order item by id")
	ErrChangeStockAndReserved = errors.New("Failed to change stock and reserved")
	ErrNoOrders               = errors.New("User does not have orders")
	ErrOrderNotFound          = errors.New("Order not found")
	ErrOrderItemNotFound      = errors.New("Order item not found")
	ErrCartNotFound           = errors.New("Cart not found")
	ErrEmptyCart              = errors.New("Cart is empty")
	ErrNotYourOrder           = errors.New("It's not your order")
	ErrProductAlreadyInCart   = errors.New("This product is already in the cart")
	ErrSellerNotSet           = errors.New("Seller is not set for this order")
	ErrOrderStatusInvalid     = errors.New("Invalid order status")
	ErrInsufficientStock      = errors.New("Insufficient stock to ship order")

	// payments
	ErrPaymentExpired       = errors.New("Payment period has expired")
	ErrPaymentDeclined      = errors.New("Payment was declined by bank")
	ErrInvalidPaymentAmount = errors.New("Payment amount does not match order total")

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
	ErrProductDeleted         = errors.New("Product is deleted")
	ErrStockBelowReserved     = errors.New("Stock quantity cannot be less than reserved quantity")

	// review
	ErrCreateReview          = errors.New("Failed to create review")
	ErrGetReviewByID         = errors.New("Failed to get review by id")
	ErrUpdateReview          = errors.New("Failed to update review")
	ErrDeleteReviewByID      = errors.New("Failed to delete review")
	ErrGetReviewsByProductID = errors.New("Failed to get reviews by product id")
	ErrNotYourReview         = errors.New("It's not your review")
	ErrReviewNotFound        = errors.New("Review not found")
	ErrReviewAlreadyExists   = errors.New("You have already reviewed this product")

	// backoffice
	ErrGetPlatformStats = errors.New("Failed to get platform stats")
)

// For repos usage
var (
	ErrNotFound               = errors.New("Not found")
	ErrNoChangesInUpdate      = errors.New("All update fields are nil")
	ErrMustBeInTransaction    = errors.New("This method must be called in transaction")
	ErrStockInvariantViolated = errors.New("Stock invariant violated: reserved or stock quantity out of bounds")
)
