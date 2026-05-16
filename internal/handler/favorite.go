package handler

import (
	"net/http"
	"strconv"

	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
)

type FavoriteHandler struct {
	favoriteService service.FavoriteService
}

func NewFavoriteHandler(favoriteSvc service.FavoriteService) FavoriteHandler {
	return FavoriteHandler{favoriteService: favoriteSvc}
}

// PUT /api/v1/favorites/{productID} — idempotent.
// 201 on first add, 200 on repeat.
func (fh FavoriteHandler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := favoriteProductID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := fh.favoriteService.AddFavorite(r.Context(), actor, id)
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

// DELETE /api/v1/favorites/{productID} — idempotent, always 204.
func (fh FavoriteHandler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := favoriteProductID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := fh.favoriteService.RemoveFavorite(r.Context(), actor, id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/favorites/{productID} — favorite-state probe for one product.
func (fh FavoriteHandler) GetFavoriteState(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := favoriteProductID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	state, err := fh.favoriteService.IsProductFavorite(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, favoriteStateDTO(state))
}

// GET /api/v1/favorites — list current user's favorite products.
func (fh FavoriteHandler) GetFavoriteProducts(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	pg, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	products, err := fh.favoriteService.GetFavoriteProducts(r.Context(), actor, pg)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	dtos := make([]ProductDTO, len(products))
	for i, p := range products {
		dtos[i] = productDTO(p)
	}
	writeJSON(w, http.StatusOK, dtos)
}

func favoriteProductID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil || id < 1 {
		return 0, ErrInvalidIDParam
	}
	return id, nil
}
