package service

import (
	"errors"
)

// service layer errors
var (
	// users
	ErrAccWithEmailCheck      = errors.New("Error occured when check account with such email")
	ErrAccountWithEmailExists = errors.New("Account with this email already exists")
	ErrHashingPassword        = errors.New("Error occured when hashing passowrd")
	ErrCreateUser             = errors.New("Error occured when creating user")
	ErrUpdateUser             = errors.New("Error occured when updating user")
	ErrGetUserByID            = errors.New("Error occured when getting user by id")
	ErrGetUserByEmail         = errors.New("Error occured when getting user by email")
	ErrDeleteUser             = errors.New("Error occured when deleting user")
	ErrChangePasswordUser     = errors.New("Error occured when changing user password")

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
	ErrCartNotFound           = errors.New("Cart not found")
	ErrEmptyCart              = errors.New("Cart is empty")
	ErrNotYourOrder           = errors.New("It's not your order")
	ErrSellerNotSet           = errors.New("Seller is not set for this order")
	ErrOrderStatusInvalid     = errors.New("Invalid order status")

	//categories
	

	// products
	ErrGetProductByID = errors.New("Failed to get product by id")
	ErrQuantityTooBig = errors.New("Quantity too big, stock is less than that")
)

// For repos usage
var (
	ErrNotFound          = errors.New("Not found")
	ErrNoChangesInUpdate = errors.New("All update fields are nil")
)
