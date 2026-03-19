package handler

import (
	"errors"
	"fmt"
)

// Validation
var (
	// RegisterRequest validation
	ErrFullNameShouldBeTwoWords = errors.New("Full name should consist of two words")
	ErrPasswordTooShort         = fmt.Errorf("Password too short. Should be at least %d symbols", minPassLen)
	ErrInvalidRole              = errors.New("Invalid role name")
	ErrRoleUnavailable          = errors.New("User with this role is unavailable to be created")
	ErrInvalidEmail             = errors.New("Invalid email")
	ErrInvalidPhone             = errors.New("Invalid phone number")

	// UpdateProfileRequest validation
	ErrUpdateProfileAllNil = errors.New("No updates in body")

	// SellerHandler validation
	ErrCompanyNameRequired = errors.New("Company name is required")
	ErrUpdateSellerAllNil  = errors.New("No updates in body")

	// ProductHandler validation
	ErrProductNameRequired = errors.New("Product name is required")
	ErrProductPriceInvalid = errors.New("Product price must be greater than zero")
	ErrProductStockInvalid = errors.New("Stock quantity cannot be negative")
	ErrUpdateProductAllNil = errors.New("No updates in body")

	// ProductHandler / catalog query validation
	ErrInvalidMinPriceOption        = errors.New("Invalid min_price value")
	ErrInvalidMaxPriceOption        = errors.New("Invalid max_price value")
	ErrMaxIsLessThanMin             = errors.New("max_price must be greater than or equal to min_price")
	ErrNoPageInPaginationOptions    = errors.New("limit provided without page")
	ErrNoLimitInPaginationOptions   = errors.New("page provided without limit")
	ErrInvalidPagePaginationOption  = errors.New("Invalid page value, must be a positive integer")
	ErrInvalidLimitPaginationOption = errors.New("Invalid limit value, must be a positive integer")
	ErrInvalidSortingOrder          = errors.New("Invalid sorting_order, expected 'asc' or 'desc'")

	// CategoryHandler validation
	ErrCategoryNameRequired  = errors.New("Category name is required")
	ErrUpdateCategoryAllNil  = errors.New("No updates in body")

	// AddressHandler validation
	ErrAddressCityRequired    = errors.New("City is required")
	ErrAddressStreetRequired  = errors.New("Street is required")
	ErrAddressZipCodeRequired = errors.New("Zip code is required")
	ErrUpdateAddressAllNil    = errors.New("No updates in body")

	// ReviewHandler validation
	ErrReviewRatingInvalid = errors.New("Rating must be between 1 and 5")
	ErrUpdateReviewAllNil  = errors.New("No updates in body")

	// OrderHandler validation
	ErrCartQuantityInvalid = errors.New("Quantity must be greater than zero")

	// Common param validation
	ErrInvalidIDParam             = errors.New("Invalid ID parameter")
	ErrInvalidDateFormat          = errors.New("Invalid date format, expected YYYY-MM-DD")
	ErrDateFromRequired           = errors.New("date_from query parameter is required")
	ErrDateFromMustBeBeforeDateTo = errors.New("date_from must be before date_to")
	ErrDateToRequired             = errors.New("date_to query parameter is required")
)

// other errors
var (
	ErrInternalServer = errors.New("Internal server error")

	ErrDecodeFailed         = errors.New("Decode failed")
	ErrTokenClaimsGetFailed = errors.New("Failed to get token claims from request context")
)
