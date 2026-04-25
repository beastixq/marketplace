package handler

import (
	"net/http"

	"github.com/beastixq/marketplace/internal/middleware"
	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
)

func actorFromClaims(claims model.TokenClaims) service.Actor {
	return service.Actor{
		UserID: claims.UserID,
		Role:   claims.Role,
	}
}

func actorFromRequest(r *http.Request) (service.Actor, bool) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		return service.Actor{}, false
	}
	return actorFromClaims(claims), true
}
