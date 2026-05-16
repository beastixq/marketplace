package web

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/go-chi/chi/v5"
)

// --- Favorites ---

const favoritesPerPage = 12

type favoritesData struct {
	Products []model.Product
	Page     int
	HasMore  bool
	User     *userInfo
	Notice   string
}

func (fd favoritesData) PaginationURL(page int) string {
	v := url.Values{}
	v.Set("page", strconv.Itoa(page))
	return "/favorites?" + v.Encode()
}

func (wh *WebHandler) Favorites(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	products, err := wh.favoriteService.GetFavoriteProducts(r.Context(), user.actor(), model.PaginationOpts{Page: page, Limit: favoritesPerPage})
	if err != nil {
		log.Printf("Favorites list error: %v", err)
		http.Error(w, "Failed to load favorites", http.StatusInternalServerError)
		return
	}

	wh.render(w, "favorites", favoritesData{
		Products: products,
		Page:     page,
		HasMore:  len(products) == favoritesPerPage,
		User:     user,
		Notice:   r.URL.Query().Get("notice"),
	})
}

func (wh *WebHandler) FavoriteAdd(w http.ResponseWriter, r *http.Request) {
	wh.favoriteMutate(w, r, true)
}

func (wh *WebHandler) FavoriteRemove(w http.ResponseWriter, r *http.Request) {
	wh.favoriteMutate(w, r, false)
}

func (wh *WebHandler) favoriteMutate(w http.ResponseWriter, r *http.Request, add bool) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil || productID < 1 {
		http.Redirect(w, r, "/favorites", http.StatusSeeOther)
		return
	}

	actor := user.actor()
	if add {
		if _, err := wh.favoriteService.AddFavorite(r.Context(), actor, productID); err != nil {
			log.Printf("FavoriteAdd error: %v", err)
		}
	} else {
		if err := wh.favoriteService.RemoveFavorite(r.Context(), actor, productID); err != nil {
			log.Printf("FavoriteRemove error: %v", err)
		}
	}

	http.Redirect(w, r, favoriteRedirectTarget(r, productID, add), http.StatusSeeOther)
}

// favoriteRedirectTarget returns the post-mutation redirect URL. It honors a
// same-origin return_to form value and falls back to a sensible default:
// product page on add, favorites list on remove.
func favoriteRedirectTarget(r *http.Request, productID int64, add bool) string {
	if rt := r.FormValue("return_to"); strings.HasPrefix(rt, "/") && !strings.HasPrefix(rt, "//") {
		return rt
	}
	if add {
		return fmt.Sprintf("/products/%d", productID)
	}
	return "/favorites?notice=removed"
}
