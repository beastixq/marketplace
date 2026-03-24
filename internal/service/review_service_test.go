package service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"go.uber.org/mock/gomock"
)

var someReviewComment = "Great product!"
var someReviewRating int8 = 5

var someReview = m.Review{
	ID:        someID,
	UserID:    someID,
	ProductID: someID,
	Rating:    someReviewRating,
	Comment:   &someReviewComment,
	CreatedAt: someTime,
}

type MockReviewReturn struct {
	Review m.Review
	Error  error
}

func assertReview(t *testing.T, got, want m.Review) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid review. expected: %v, got: %v", want, got)
	}
}

func TestGetReviewByID(t *testing.T) {
	mock := mock_service.NewMockReviewRepo(gomock.NewController(t))
	svc := service.NewReviewService(mock)
	ctx := context.Background()

	type testCase struct {
		Description    string
		ReviewID       int64
		MockReturn     MockReviewReturn
		ExpectedReview m.Review
		ExpectedErr    error
	}

	tCases := []testCase{
		{
			Description:    "Success",
			ReviewID:       someID,
			MockReturn:     MockReviewReturn{Review: someReview},
			ExpectedReview: someReview,
		},
		{
			Description: "Review not found",
			ReviewID:    123,
			MockReturn:  MockReviewReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrReviewNotFound,
		},
		{
			Description: "Repo error",
			ReviewID:    someID,
			MockReturn:  MockReviewReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetReviewByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetReviewByID(ctx, tCase.ReviewID).Return(tCase.MockReturn.Review, tCase.MockReturn.Error)
			r, err := svc.GetReviewByID(ctx, tCase.ReviewID)
			assertError(t, err, tCase.ExpectedErr)
			assertReview(t, r, tCase.ExpectedReview)
		})
	}
}

func TestCreateReview(t *testing.T) {
	mock := mock_service.NewMockReviewRepo(gomock.NewController(t))
	svc := service.NewReviewService(mock)
	ctx := context.Background()

	reviewCreate := m.ReviewCreate{
		UserID:    someID,
		ProductID: someID,
		Rating:    someReviewRating,
		Comment:   &someReviewComment,
	}

	type testCase struct {
		Description string
		Create      m.ReviewCreate
		MockReturn  MockCreateReturn
		ExpectedID  int64
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			Create:      reviewCreate,
			MockReturn:  MockCreateReturn{ID: someID},
			ExpectedID:  someID,
		},
		{
			Description: "Repo error",
			Create:      reviewCreate,
			MockReturn:  MockCreateReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrCreateReview,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().CreateReview(ctx, tCase.Create).Return(tCase.MockReturn.ID, tCase.MockReturn.Error)
			id, err := svc.CreateReview(ctx, tCase.Create)
			assertError(t, err, tCase.ExpectedErr)
			if id != tCase.ExpectedID {
				t.Fatalf("invalid id. expected: %v, got: %v", tCase.ExpectedID, id)
			}
		})
	}
}

func TestUpdateReview(t *testing.T) {
	mock := mock_service.NewMockReviewRepo(gomock.NewController(t))
	svc := service.NewReviewService(mock)
	ctx := context.Background()

	newRating := int8(3)
	newComment := "Updated comment"
	reviewUpdate := m.ReviewUpdate{
		Rating:  &newRating,
		Comment: &newComment,
	}

	updatedReview := m.Review{
		ID:        someReview.ID,
		UserID:    someReview.UserID,
		ProductID: someReview.ProductID,
		Rating:    newRating,
		Comment:   &newComment,
		CreatedAt: someReview.CreatedAt,
	}

	type testCase struct {
		Description    string
		UserID         int64
		ReviewID       int64
		Update         m.ReviewUpdate
		MockGet        MockReviewReturn
		MockUpdate     *MockReviewReturn
		ExpectedReview m.Review
		ExpectedErr    error
	}

	tCases := []testCase{
		{
			Description:    "Success",
			UserID:         someID,
			ReviewID:       someID,
			Update:         reviewUpdate,
			MockGet:        MockReviewReturn{Review: someReview},
			MockUpdate:     &MockReviewReturn{Review: updatedReview},
			ExpectedReview: updatedReview,
		},
		{
			Description: "Review not found",
			UserID:      someID,
			ReviewID:    123,
			Update:      reviewUpdate,
			MockGet:     MockReviewReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrReviewNotFound,
		},
		{
			Description: "GetReviewByID repo error",
			UserID:      someID,
			ReviewID:    someID,
			Update:      reviewUpdate,
			MockGet:     MockReviewReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetReviewByID,
		},
		{
			Description: "Not your review",
			UserID:      otherUserID,
			ReviewID:    someID,
			Update:      reviewUpdate,
			MockGet:     MockReviewReturn{Review: someReview},
			ExpectedErr: service.ErrNotYourReview,
		},
		{
			Description: "UpdateReview repo error",
			UserID:      someID,
			ReviewID:    someID,
			Update:      reviewUpdate,
			MockGet:     MockReviewReturn{Review: someReview},
			MockUpdate:  &MockReviewReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrUpdateReview,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetReviewByID(ctx, tCase.ReviewID).Return(tCase.MockGet.Review, tCase.MockGet.Error)
			if tCase.MockUpdate != nil {
				mock.EXPECT().UpdateReview(ctx, tCase.ReviewID, tCase.Update).Return(tCase.MockUpdate.Review, tCase.MockUpdate.Error)
			}
			r, err := svc.UpdateReview(ctx, tCase.UserID, tCase.ReviewID, tCase.Update)
			assertError(t, err, tCase.ExpectedErr)
			assertReview(t, r, tCase.ExpectedReview)
		})
	}
}

func TestDeleteReviewByID(t *testing.T) {
	mock := mock_service.NewMockReviewRepo(gomock.NewController(t))
	svc := service.NewReviewService(mock)
	ctx := context.Background()

	type testCase struct {
		Description string
		UserID      int64
		ReviewID    int64
		MockGet     MockReviewReturn
		MockDelete  *error
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			UserID:      someID,
			ReviewID:    someID,
			MockGet:     MockReviewReturn{Review: someReview},
			MockDelete:  ptrErr(nil),
		},
		{
			Description: "Review not found",
			UserID:      someID,
			ReviewID:    123,
			MockGet:     MockReviewReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrReviewNotFound,
		},
		{
			Description: "GetReviewByID repo error",
			UserID:      someID,
			ReviewID:    someID,
			MockGet:     MockReviewReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetReviewByID,
		},
		{
			Description: "Not your review",
			UserID:      otherUserID,
			ReviewID:    someID,
			MockGet:     MockReviewReturn{Review: someReview},
			ExpectedErr: service.ErrNotYourReview,
		},
		{
			Description: "DeleteReviewByID repo error",
			UserID:      someID,
			ReviewID:    someID,
			MockGet:     MockReviewReturn{Review: someReview},
			MockDelete:  ptrErr(errors.New("some repo error")),
			ExpectedErr: service.ErrDeleteReviewByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetReviewByID(ctx, tCase.ReviewID).Return(tCase.MockGet.Review, tCase.MockGet.Error)
			if tCase.MockDelete != nil {
				mock.EXPECT().DeleteReviewByID(ctx, tCase.ReviewID).Return(*tCase.MockDelete)
			}
			err := svc.DeleteReviewByID(ctx, tCase.UserID, tCase.ReviewID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}
