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
	pg, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	categories, err := ch.categoryService.GetCategories(r.Context(), pg)
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := ch.categoryService.CreateCategory(r.Context(), actor, model.CategoryCreate{
		ParentID:    req.ParentID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			writeError(w, http.StatusForbidden, service.ErrPermissionDenied.Error())
			return
		}
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

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

	category, err := ch.categoryService.UpdateCategory(r.Context(), actor, id, model.CategoryUpdate{
		ParentID:    req.ParentID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			writeError(w, http.StatusForbidden, service.ErrPermissionDenied.Error())
			return
		}
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := ch.categoryService.DeleteCategoryByID(r.Context(), actor, id); err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			writeError(w, http.StatusForbidden, service.ErrPermissionDenied.Error())
			return
		}
		if errors.Is(err, service.ErrCategoryNotFound) {
			writeError(w, http.StatusNotFound, service.ErrCategoryNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
