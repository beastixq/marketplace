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

// POST /api/v1/favorites/{productID}
func (fh FavoriteHandler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := fh.favoriteService.AddFavorite(r.Context(), actor, productID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/favorites/{productID}
func (fh FavoriteHandler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := fh.favoriteService.RemoveFavorite(r.Context(), actor, productID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/favorites
func (fh FavoriteHandler) GetMyFavorites(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	pagination, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	favorites, err := fh.favoriteService.GetMyFavorites(r.Context(), actor, pagination)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	result := make([]FavoriteDTO, len(favorites))
	for i := range favorites {
		result[i] = favoriteDTO(favorites[i])
	}
	writeJSON(w, http.StatusOK, result)
}
