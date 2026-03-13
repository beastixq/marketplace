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

const claimsKey contextKey = "claims"

func ClaimsFromCtx(ctx context.Context) (claims m.TokenClaims, ok bool) {
	value := ctx.Value(claimsKey)
	if claims, ok = value.(m.TokenClaims); !ok {
		return m.TokenClaims{}, false
	}
	return claims, true
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

			claims, err := authService.ValidateToken(token)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
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
