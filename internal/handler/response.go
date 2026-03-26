package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/beastixq/marketplace/internal/model"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Add("Content-type", "application/json")
	body, _ := json.Marshal(payload)
	w.WriteHeader(status)
	w.Write(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-type", "application/json")
	body, _ := json.Marshal(map[string]string{"error": message})
	w.WriteHeader(status)
	w.Write(body)
}

func parsePagination(r *http.Request) (model.PaginationOpts, error) {
	q := r.URL.Query()
	if !q.Has("page") && !q.Has("limit") {
		return model.PaginationOpts{}, nil
	}
	if q.Has("page") && !q.Has("limit") {
		return model.PaginationOpts{}, ErrNoLimitInPaginationOptions
	}
	if !q.Has("page") && q.Has("limit") {
		return model.PaginationOpts{}, ErrNoPageInPaginationOptions
	}
	page, err := strconv.Atoi(q.Get("page"))
	if err != nil || page < 1 {
		return model.PaginationOpts{}, ErrInvalidPagePaginationOption
	}
	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit < 1 {
		return model.PaginationOpts{}, ErrInvalidLimitPaginationOption
	}
	return model.PaginationOpts{Page: page, Limit: limit}, nil
}
