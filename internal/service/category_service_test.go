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

var someCategoryDescription = "Electronics and gadgets"
var someParentID int64 = 10

var someCategory = m.Category{
	ID:          someID,
	ParentID:    &someParentID,
	Name:        "Smartphones",
	Description: &someCategoryDescription,
}

type MockCategoryReturn struct {
	Category m.Category
	Error    error
}

func assertCategory(t *testing.T, got, want m.Category) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid category. expected: %v, got: %v", want, got)
	}
}

func TestGetCategories(t *testing.T) {
	mock := mock_service.NewMockCategoryRepo(gomock.NewController(t))
	svc := service.NewCategoryService(mock)
	ctx := context.Background()

	categories := []m.Category{someCategory}
	opts := m.PaginationOpts{Page: 1, Limit: 10}

	type testCase struct {
		Description        string
		Opts               m.PaginationOpts
		MockCategories     []m.Category
		MockError          error
		ExpectedCategories []m.Category
		ExpectedErr        error
	}

	tCases := []testCase{
		{
			Description:        "Success",
			Opts:               opts,
			MockCategories:     categories,
			ExpectedCategories: categories,
		},
		{
			Description: "Success empty list",
			Opts:        opts,
		},
		{
			Description: "Repo error",
			Opts:        opts,
			MockError:   errors.New("some repo error"),
			ExpectedErr: service.ErrGetCategories,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetCategories(ctx, tCase.Opts).Return(tCase.MockCategories, tCase.MockError)
			cats, err := svc.GetCategories(ctx, tCase.Opts)
			assertError(t, err, tCase.ExpectedErr)
			if !reflect.DeepEqual(cats, tCase.ExpectedCategories) {
				t.Fatalf("invalid categories. expected: %v, got: %v", tCase.ExpectedCategories, cats)
			}
		})
	}
}

func TestGetCategoryByID(t *testing.T) {
	mock := mock_service.NewMockCategoryRepo(gomock.NewController(t))
	svc := service.NewCategoryService(mock)
	ctx := context.Background()

	type testCase struct {
		Description      string
		CategoryID       int64
		MockReturn       MockCategoryReturn
		ExpectedCategory m.Category
		ExpectedErr      error
	}

	tCases := []testCase{
		{
			Description:      "Success",
			CategoryID:       someID,
			MockReturn:       MockCategoryReturn{Category: someCategory},
			ExpectedCategory: someCategory,
		},
		{
			Description: "Category not found",
			CategoryID:  123,
			MockReturn:  MockCategoryReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrCategoryNotFound,
		},
		{
			Description: "Repo error",
			CategoryID:  someID,
			MockReturn:  MockCategoryReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetCategoryByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetCategoryByID(ctx, tCase.CategoryID).Return(tCase.MockReturn.Category, tCase.MockReturn.Error)
			cat, err := svc.GetCategoryByID(ctx, tCase.CategoryID)
			assertError(t, err, tCase.ExpectedErr)
			assertCategory(t, cat, tCase.ExpectedCategory)
		})
	}
}

func TestCreateCategory(t *testing.T) {
	mock := mock_service.NewMockCategoryRepo(gomock.NewController(t))
	svc := service.NewCategoryService(mock)
	ctx := context.Background()

	someCategoryCreate := m.CategoryCreate{
		ParentID:    &someParentID,
		Name:        "Smartphones",
		Description: &someCategoryDescription,
	}

	type testCase struct {
		Description string
		Create      m.CategoryCreate
		MockReturn  MockCreateReturn
		ExpectedID  int64
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			Create:      someCategoryCreate,
			MockReturn:  MockCreateReturn{ID: someID},
			ExpectedID:  someID,
		},
		{
			Description: "Repo error",
			Create:      someCategoryCreate,
			MockReturn:  MockCreateReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrCreateCategory,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().CreateCategory(ctx, tCase.Create).Return(tCase.MockReturn.ID, tCase.MockReturn.Error)
			id, err := svc.CreateCategory(ctx, testActor(someID, m.RoleAdmin), tCase.Create)
			assertError(t, err, tCase.ExpectedErr)
			if id != tCase.ExpectedID {
				t.Fatalf("invalid id. expected: %v, got: %v", tCase.ExpectedID, id)
			}
		})
	}
}

func TestUpdateCategory(t *testing.T) {
	mock := mock_service.NewMockCategoryRepo(gomock.NewController(t))
	svc := service.NewCategoryService(mock)
	ctx := context.Background()

	newName := "Tablets"
	categoryUpdate := m.CategoryUpdate{
		Name: &newName,
	}

	updatedCategory := m.Category{
		ID:          someCategory.ID,
		ParentID:    someCategory.ParentID,
		Name:        newName,
		Description: someCategory.Description,
	}

	type testCase struct {
		Description      string
		CategoryID       int64
		Update           m.CategoryUpdate
		MockReturn       MockCategoryReturn
		ExpectedCategory m.Category
		ExpectedErr      error
	}

	tCases := []testCase{
		{
			Description:      "Success",
			CategoryID:       someID,
			Update:           categoryUpdate,
			MockReturn:       MockCategoryReturn{Category: updatedCategory},
			ExpectedCategory: updatedCategory,
		},
		{
			Description: "Repo error",
			CategoryID:  someID,
			Update:      categoryUpdate,
			MockReturn:  MockCategoryReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrUpdateCategory,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().UpdateCategory(ctx, tCase.CategoryID, tCase.Update).Return(tCase.MockReturn.Category, tCase.MockReturn.Error)
			cat, err := svc.UpdateCategory(ctx, testActor(someID, m.RoleAdmin), tCase.CategoryID, tCase.Update)
			assertError(t, err, tCase.ExpectedErr)
			assertCategory(t, cat, tCase.ExpectedCategory)
		})
	}
}

func TestDeleteCategoryByID(t *testing.T) {
	mock := mock_service.NewMockCategoryRepo(gomock.NewController(t))
	svc := service.NewCategoryService(mock)
	ctx := context.Background()

	type testCase struct {
		Description string
		CategoryID  int64
		MockError   error
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			CategoryID:  someID,
		},
		{
			Description: "Repo error",
			CategoryID:  someID,
			MockError:   errors.New("some repo error"),
			ExpectedErr: service.ErrDeleteCategory,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().DeleteCategoryByID(ctx, tCase.CategoryID).Return(tCase.MockError)
			err := svc.DeleteCategoryByID(ctx, testActor(someID, m.RoleAdmin), tCase.CategoryID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}
