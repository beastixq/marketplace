package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/beastixq/marketplace/internal/middleware"
	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
)

type ReviewHandler struct {
	reviewService service.ReviewService
}

func NewReviewHandler(reviewSvc service.ReviewService) ReviewHandler {
	return ReviewHandler{reviewService: reviewSvc}
}

type CreateReviewRequest struct {
	ProductID int64   `json:"product_id"`
	Rating    int8    `json:"rating"`
	Comment   *string `json:"comment"`
}

func (cr CreateReviewRequest) Validate() error {
	if cr.Rating < 1 || cr.Rating > 5 {
		return ErrReviewRatingInvalid
	}
	return nil
}

// POST /api/v1/reviews
func (rh ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	var req CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := rh.reviewService.CreateReview(r.Context(), model.ReviewCreate{
		UserID:    claims.UserID,
		ProductID: req.ProductID,
		Rating:    req.Rating,
		Comment:   req.Comment,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

type UpdateReviewRequest struct {
	Rating  *int8   `json:"rating"`
	Comment *string `json:"comment"`
}

func (ur UpdateReviewRequest) Validate() error {
	if ur.Rating == nil && ur.Comment == nil {
		return ErrUpdateReviewAllNil
	}
	if ur.Rating != nil && (*ur.Rating < 1 || *ur.Rating > 5) {
		return ErrReviewRatingInvalid
	}
	return nil
}

// PATCH /api/v1/reviews/:id
func (rh ReviewHandler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	var req UpdateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	review, err := rh.reviewService.UpdateReview(r.Context(), claims.UserID, id, model.ReviewUpdate{
		Rating:  req.Rating,
		Comment: req.Comment,
	})
	if err != nil {
		if errors.Is(err, service.ErrReviewNotFound) {
			writeError(w, http.StatusNotFound, service.ErrReviewNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourReview) {
			writeError(w, http.StatusForbidden, service.ErrNotYourReview.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, reviewFromService(review))
}

// DELETE /api/v1/reviews/:id
func (rh ReviewHandler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := rh.reviewService.DeleteReviewByID(r.Context(), claims.UserID, id); err != nil {
		if errors.Is(err, service.ErrReviewNotFound) {
			writeError(w, http.StatusNotFound, service.ErrReviewNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourReview) {
			writeError(w, http.StatusForbidden, service.ErrNotYourReview.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
