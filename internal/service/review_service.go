package service

import (
	"context"
	"errors"
	"fmt"

	m "github.com/beastixq/marketplace/internal/model"
)

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_review_repo.go github.com/beastixq/marketplace/internal/service ReviewRepo
type ReviewRepo interface {
	GetReviewByID(ctx context.Context, id int64) (r m.Review, err error)
	GetReviewsByProductID(ctx context.Context, pid int64, opts m.PaginationOpts) (rs []m.Review, err error)
	CreateReview(ctx context.Context, rc m.ReviewCreate) (id int64, err error)
	UpdateReview(ctx context.Context, id int64, ru m.ReviewUpdate) (r m.Review, err error)
	DeleteReviewByID(ctx context.Context, id int64) (err error)
}

type ReviewPurchaseChecker interface {
	UserPurchasedProduct(ctx context.Context, userID int64, productID int64) (bool, error)
}

// ReviewProductGetter is the narrow product lookup ReviewService needs to
// reject reviews for soft-deleted products.
type ReviewProductGetter interface {
	GetProductByID(ctx context.Context, id int64) (m.Product, error)
}

type ReviewService struct {
	reviewRepo      ReviewRepo
	purchaseChecker ReviewPurchaseChecker
	productGetter   ReviewProductGetter
}

func NewReviewService(reviewRepo ReviewRepo, purchaseChecker ReviewPurchaseChecker, productGetter ReviewProductGetter) ReviewService {
	return ReviewService{reviewRepo: reviewRepo, purchaseChecker: purchaseChecker, productGetter: productGetter}
}

func (rs ReviewService) GetReviewByID(ctx context.Context, id int64) (r m.Review, err error) {
	r, err = rs.reviewRepo.GetReviewByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Review{}, ErrReviewNotFound
		}
		return m.Review{}, fmt.Errorf("%w: %v", ErrGetReviewByID, err)
	}
	return r, nil
}

func (rs ReviewService) CreateReview(ctx context.Context, actor Actor, rc m.ReviewCreate) (id int64, err error) {
	if !actor.HasRole(m.RoleBuyer) {
		return 0, ErrPermissionDenied
	}

	rc.UserID = actor.UserID

	// Block reviews on soft-deleted products even when the buyer purchased
	// them earlier. Existing reviews remain readable; we just refuse to
	// accept new content tied to a product that is no longer offered.
	product, err := rs.productGetter.GetProductByID(ctx, rc.ProductID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return 0, ErrProductNotFound
		}
		return 0, fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}
	if product.DeletedAt != nil {
		return 0, ErrProductDeleted
	}

	purchased, err := rs.purchaseChecker.UserPurchasedProduct(ctx, actor.UserID, rc.ProductID)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrCheckReviewPurchase, err)
	}
	if !purchased {
		return 0, ErrReviewPurchaseRequired
	}

	id, err = rs.reviewRepo.CreateReview(ctx, rc)
	if err != nil {
		if errors.Is(err, ErrReviewAlreadyExists) {
			return 0, ErrReviewAlreadyExists
		}
		return 0, fmt.Errorf("%w: %v", ErrCreateReview, err)
	}
	return id, nil
}

func (rs ReviewService) UpdateReview(ctx context.Context, actor Actor, id int64, ru m.ReviewUpdate) (r m.Review, err error) {
	if !actor.HasRole(m.RoleBuyer) {
		return m.Review{}, ErrPermissionDenied
	}

	r, err = rs.reviewRepo.GetReviewByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Review{}, ErrReviewNotFound
		}
		return m.Review{}, fmt.Errorf("%w: %v", ErrGetReviewByID, err)
	}
	if r.UserID != actor.UserID {
		return m.Review{}, ErrNotYourReview
	}

	r, err = rs.reviewRepo.UpdateReview(ctx, id, ru)
	if err != nil {
		return m.Review{}, fmt.Errorf("%w: %v", ErrUpdateReview, err)
	}
	return r, nil
}

func (rs ReviewService) DeleteReviewByID(ctx context.Context, actor Actor, id int64) (err error) {
	r, err := rs.reviewRepo.GetReviewByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrReviewNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetReviewByID, err)
	}
	if !actor.IsAdmin() {
		if !actor.HasRole(m.RoleBuyer) {
			return ErrPermissionDenied
		}
		if r.UserID != actor.UserID {
			return ErrNotYourReview
		}
	}

	err = rs.reviewRepo.DeleteReviewByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteReviewByID, err)
	}
	return nil
}
