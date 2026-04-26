package middleware

import (
	"context"
	"net/http"
	"slices"
	"strings"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
)

type contextKey string

const (
	claimsKey      contextKey = "claims"
	actorHolderKey contextKey = "actorHolder"
)

// actorHolder is a request-scoped mutable cell. AuthMiddleware writes
// claims into it; outer middleware (e.g. RequestLogger) reads them after
// the inner chain returns. A pointer is required because Request.Context
// substitution only propagates down the chain, not back up.
type actorHolder struct {
	Claims *m.TokenClaims
}

func ClaimsFromCtx(ctx context.Context) (claims m.TokenClaims, ok bool) {
	value := ctx.Value(claimsKey)
	if claims, ok = value.(m.TokenClaims); !ok {
		return m.TokenClaims{}, false
	}
	return claims, true
}

// ActorFromHolder reads the authenticated claims captured by AuthMiddleware
// for the current request. Returns false if the request is unauthenticated
// or the holder middleware was not installed.
func ActorFromHolder(ctx context.Context) (m.TokenClaims, bool) {
	h, ok := ctx.Value(actorHolderKey).(*actorHolder)
	if !ok || h == nil || h.Claims == nil {
		return m.TokenClaims{}, false
	}
	return *h.Claims, true
}

// ActorHolder installs a request-scoped holder so that downstream auth
// middleware can publish the authenticated actor to outer middleware.
func ActorHolder() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := &actorHolder{}
			ctx := context.WithValue(r.Context(), actorHolderKey, h)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AuthMiddleware(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			authType, token, ok := strings.Cut(header, " ")
			if !ok || authType != "Bearer" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			claims, err := authService.ValidateToken(r.Context(), token)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			if h, ok := ctx.Value(actorHolderKey).(*actorHolder); ok && h != nil {
				h.Claims = &claims
			}
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func RequireRole(roles ...m.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			claims, ok := ClaimsFromCtx(ctx)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !slices.Contains(roles, claims.Role) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
