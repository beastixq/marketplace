package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
)

type SellerRepo interface {
	GetSellerByID(ctx context.Context, id int64) (s m.Seller, err error)
	GetSellerByUserID(ctx context.Context, userID int64) (s m.Seller, err error)
	GetSellerStats(ctx context.Context, sellerID int64, dateFrom time.Time, dateTo time.Time) (ss m.SellerStats, err error)
	CreateSeller(ctx context.Context, sc m.SellerCreate) (id int64, err error)
	UpdateSeller(ctx context.Context, id int64, su m.SellerUpdate) (s m.Seller, err error)
	DeleteSellerByID(ctx context.Context, id int64) (err error)
}

type SellerService struct {
	sellerRepo SellerRepo
}

func NewSellerService(sellerRepo SellerRepo) SellerService {
	return SellerService{sellerRepo: sellerRepo}
}

func (ss SellerService) GetSellerByID(ctx context.Context, id int64) (s m.Seller, err error) {
	s, err = ss.sellerRepo.GetSellerByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Seller{}, ErrSellerNotFound
		}
		return m.Seller{}, fmt.Errorf("%w: %v", ErrGetSellerByID, err)
	}
	return s, nil
}

func (ss SellerService) CreateSeller(ctx context.Context, sc m.SellerCreate) (id int64, err error) {
	id, err = ss.sellerRepo.CreateSeller(ctx, sc)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrCreateSeller, err)
	}
	return id, nil
}

func (ss SellerService) UpdateSeller(ctx context.Context, userID, id int64, su m.SellerUpdate) (s m.Seller, err error) {
	s, err = ss.sellerRepo.GetSellerByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Seller{}, ErrSellerNotFound
		}
		return m.Seller{}, fmt.Errorf("%w: %v", ErrGetSellerByID, err)
	}
	if s.UserID != userID {
		return m.Seller{}, ErrNotYourSeller
	}

	s, err = ss.sellerRepo.UpdateSeller(ctx, id, su)
	if err != nil {
		return m.Seller{}, fmt.Errorf("%w: %v", ErrUpdateSeller, err)
	}
	return s, nil
}

func (ss SellerService) DeleteSellerByID(ctx context.Context, userID, id int64) (err error) {
	s, err := ss.sellerRepo.GetSellerByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrSellerNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetSellerByID, err)
	}
	if s.UserID != userID {
		return ErrNotYourSeller
	}

	err = ss.sellerRepo.DeleteSellerByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteSellerByID, err)
	}
	return nil
}

func (ssvc SellerService) GetSellerStats(ctx context.Context, userID, sellerID int64, dateFrom time.Time, dateTo time.Time) (ss m.SellerStats, err error) {
	s, err := ssvc.sellerRepo.GetSellerByID(ctx, sellerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.SellerStats{}, ErrSellerNotFound
		}
		return m.SellerStats{}, fmt.Errorf("%w: %v", ErrGetSellerByID, err)
	}
	if s.UserID != userID {
		return m.SellerStats{}, ErrNotYourSeller
	}

	ss, err = ssvc.sellerRepo.GetSellerStats(ctx, sellerID, dateFrom, dateTo)
	if err != nil {
		return m.SellerStats{}, fmt.Errorf("%w: %v", ErrGetSellerStats, err)
	}
	return ss, nil
}
