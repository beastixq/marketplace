package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

type FavoriteHandler struct {
	favoriteService service.FavoriteService
}

func NewFavoriteHandler(favoriteSvc service.FavoriteService) FavoriteHandler {
	return FavoriteHandler{favoriteService: favoriteSvc}
}

// ----- DTOs -----

type favoriteProductDTO struct {
	ID      int64           `json:"id"`
	Name    string          `json:"name"`
	Price   decimal.Decimal `json:"price"`
	InStock bool            `json:"in_stock"`
}

type favoriteItemDTO struct {
	Product favoriteProductDTO `json:"product"`
	AddedAt time.Time          `json:"added_at"`
}

type favoritesListDTO struct {
	Items    []favoriteItemDTO `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Total    int               `json:"total"`
}

type favoriteStatusDTO struct {
	Favorited bool       `json:"favorited"`
	AddedAt   *time.Time `json:"added_at,omitempty"`
}

func productToFavoriteDTO(p model.Product) favoriteProductDTO {
	return favoriteProductDTO{
		ID:      p.ID,
		Name:    p.Name,
		Price:   p.Price,
		InStock: p.DeletedAt == nil && p.AvailableQuantity() > 0,
	}
}

// ----- Handlers -----

// PUT /api/v1/favorites/{productId}
func (fh FavoriteHandler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	productID, err := strconv.ParseInt(chi.URLParam(r, "productId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	created, err := fh.favoriteService.Add(r.Context(), actor.UserID, productID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if created {
		w.WriteHeader(http.StatusCreated)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// DELETE /api/v1/favorites/{productId}
func (fh FavoriteHandler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	productID, err := strconv.ParseInt(chi.URLParam(r, "productId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := fh.favoriteService.Remove(r.Context(), actor.UserID, productID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/favorites
func (fh FavoriteHandler) ListFavorites(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	page, pageSize, err := parseFavoritesPagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	items, total, err := fh.favoriteService.List(r.Context(), actor.UserID, page, pageSize)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	clampedPage, clampedPageSize := normalizeFavoritesPagination(page, pageSize)
	resp := favoritesListDTO{
		Items:    make([]favoriteItemDTO, 0, len(items)),
		Page:     clampedPage,
		PageSize: clampedPageSize,
		Total:    total,
	}
	for _, f := range items {
		resp.Items = append(resp.Items, favoriteItemDTO{
			Product: productToFavoriteDTO(f.Product),
			AddedAt: f.AddedAt,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// GET /api/v1/favorites/{productId}
func (fh FavoriteHandler) GetFavoriteStatus(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	productID, err := strconv.ParseInt(chi.URLParam(r, "productId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	favorited, addedAt, err := fh.favoriteService.IsFavorited(r.Context(), actor.UserID, productID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, favoriteStatusDTO{
		Favorited: favorited,
		AddedAt:   addedAt,
	})
}

// ----- Helpers -----

const (
	favoritesDefaultPageSize = 20
	favoritesMaxPageSize     = 100
)

// parseFavoritesPagination reads page and page_size as positive integers if
// supplied. Returns the raw values (still subject to service-side clamping).
// Non-numeric values are a 400; absence is fine.
func parseFavoritesPagination(r *http.Request) (page, pageSize int, err error) {
	page = 1
	pageSize = favoritesDefaultPageSize

	if v := r.URL.Query().Get("page"); v != "" {
		p, perr := strconv.Atoi(v)
		if perr != nil {
			return 0, 0, ErrInvalidPagePaginationOption
		}
		page = p
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		ps, perr := strconv.Atoi(v)
		if perr != nil {
			return 0, 0, ErrInvalidLimitPaginationOption
		}
		pageSize = ps
	}
	return page, pageSize, nil
}

// normalizeFavoritesPagination mirrors the service-side clamp so that the
// list response reports the effective values, not the raw ones the caller
// supplied.
func normalizeFavoritesPagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = favoritesDefaultPageSize
	}
	if pageSize > favoritesMaxPageSize {
		pageSize = favoritesMaxPageSize
	}
	return page, pageSize
}
