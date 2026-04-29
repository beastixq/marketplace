package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/go-chi/chi/v5"
)

// --- Review Submission ---

func (wh *WebHandler) ReviewSubmit(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	productID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	ratingStr := r.FormValue("rating")
	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 1 || rating > 5 {
		rating = 5
	}

	comment := r.FormValue("comment")

	rc := model.ReviewCreate{
		ProductID: productID,
		Rating:    int8(rating),
	}
	if comment != "" {
		rc.Comment = &comment
	}

	if _, err = wh.reviewService.CreateReview(r.Context(), user.actor(), rc); err != nil {
		http.Redirect(w, r, fmt.Sprintf("/products/%d?review_error=%s", productID, url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/products/%d", productID), http.StatusSeeOther)
}

func (wh *WebHandler) ReviewUpdate(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	review, err := wh.reviewService.GetReviewByID(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	ratingStr := r.FormValue("rating")
	ratingRaw, err := strconv.Atoi(ratingStr)
	if err != nil || ratingRaw < 1 || ratingRaw > 5 {
		http.Redirect(w, r, fmt.Sprintf("/products/%d?review_error=%s", review.ProductID, url.QueryEscape("Rating must be between 1 and 5")), http.StatusSeeOther)
		return
	}
	rating := int8(ratingRaw)
	comment := r.FormValue("comment")

	_, err = wh.reviewService.UpdateReview(r.Context(), user.actor(), id, model.ReviewUpdate{
		Rating:  &rating,
		Comment: &comment,
	})
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/products/%d?review_error=%s", review.ProductID, url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/products/%d", review.ProductID), http.StatusSeeOther)
}

func (wh *WebHandler) ReviewDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var productID int64
	review, err := wh.reviewService.GetReviewByID(r.Context(), id)
	if err == nil {
		productID = review.ProductID
	}

	if err = wh.reviewService.DeleteReviewByID(r.Context(), user.actor(), id); err != nil {
		if productID > 0 {
			http.Redirect(w, r, fmt.Sprintf("/products/%d?review_error=%s", productID, url.QueryEscape(err.Error())), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if productID > 0 {
		http.Redirect(w, r, fmt.Sprintf("/products/%d", productID), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
