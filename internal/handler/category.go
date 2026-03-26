package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
)

type CategoryHandler struct {
	categoryService service.CategoryService
}

func NewCategoryHandler(categorySvc service.CategoryService) CategoryHandler {
	return CategoryHandler{categoryService: categorySvc}
}

// GET /api/v1/categories
func (ch CategoryHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	var opts model.PaginationOpts
	if r.URL.Query().Has("limit") && !r.URL.Query().Has("page") {
		writeError(w, http.StatusBadRequest, ErrNoPageInPaginationOptions.Error())
		return
	}
	if r.URL.Query().Has("page") && !r.URL.Query().Has("limit") {
		writeError(w, http.StatusBadRequest, ErrNoLimitInPaginationOptions.Error())
		return
	}
	if r.URL.Query().Has("page") {
		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page <= 0 {
			writeError(w, http.StatusBadRequest, ErrInvalidPagePaginationOption.Error())
			return
		}
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			writeError(w, http.StatusBadRequest, ErrInvalidLimitPaginationOption.Error())
			return
		}
		opts = model.PaginationOpts{Page: page, Limit: limit}
	}

	categories, err := ch.categoryService.GetCategories(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	result := make([]CategoryDTO, len(categories))
	for i := range categories {
		result[i] = categoryDTO(categories[i])
	}
	writeJSON(w, http.StatusOK, result)
}

type CreateCategoryRequest struct {
	ParentID    *int64  `json:"parent_id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (cr CreateCategoryRequest) Validate() error {
	if cr.Name == "" {
		return ErrCategoryNameRequired
	}
	return nil
}

// POST /api/v1/categories — admin only (enforced at router level)
func (ch CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := ch.categoryService.CreateCategory(r.Context(), model.CategoryCreate{
		ParentID:    req.ParentID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

type UpdateCategoryRequest struct {
	ParentID    *int64  `json:"parent_id"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (ur UpdateCategoryRequest) Validate() error {
	if ur.ParentID == nil && ur.Name == nil && ur.Description == nil {
		return ErrUpdateCategoryAllNil
	}
	if ur.Name != nil && *ur.Name == "" {
		return ErrCategoryNameRequired
	}
	return nil
}

// PATCH /api/v1/categories/:id — admin only
func (ch CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	category, err := ch.categoryService.UpdateCategory(r.Context(), id, model.CategoryUpdate{
		ParentID:    req.ParentID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, service.ErrCategoryNotFound) {
			writeError(w, http.StatusNotFound, service.ErrCategoryNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, categoryDTO(category))
}

// DELETE /api/v1/categories/:id — admin only
func (ch CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := ch.categoryService.DeleteCategoryByID(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrCategoryNotFound) {
			writeError(w, http.StatusNotFound, service.ErrCategoryNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
