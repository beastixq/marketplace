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

type ReviewService struct {
	reviewRepo ReviewRepo
}

func NewReviewService(reviewRepo ReviewRepo) ReviewService {
	return ReviewService{reviewRepo: reviewRepo}
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
	id, err = rs.reviewRepo.CreateReview(ctx, rc)
	if err != nil {
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
