package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// statusRecorder wraps ResponseWriter to capture the response status
// code so the request log can include it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	if sr.status != 0 {
		return
	}
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Status() int {
	if sr.status == 0 {
		return http.StatusOK
	}
	return sr.status
}

// RequestLogger logs every HTTP request once it returns. It is the single
// authoritative place that records user actions
// and request-level errors at the API boundary; service/repo layers wrap
// errors and let this middleware emit them with full context.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}

			next.ServeHTTP(rec, r)

			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.Status()),
				slog.Duration("duration", time.Since(start)),
				slog.String("remote", r.RemoteAddr),
			}
			if claims, ok := ActorFromHolder(r.Context()); ok {
				attrs = append(attrs,
					slog.Int64("user_id", claims.UserID),
					slog.String("role", string(claims.Role)),
				)
			}

			level := levelForStatus(rec.Status())
			logger.LogAttrs(r.Context(), level, "http request", toAttrSlice(attrs)...)
		})
	}
}

// Recoverer converts panics from HTTP handlers into 500 responses and records
// the panic with stack trace in the configured structured logger.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				v := recover()
				if v == nil {
					return
				}

				logger.ErrorContext(r.Context(), "http panic recovered",
					slog.Any("panic", v),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("remote", r.RemoteAddr),
					slog.String("stack", string(debug.Stack())),
				)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func levelForStatus(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

func toAttrSlice(in []any) []slog.Attr {
	out := make([]slog.Attr, 0, len(in))
	for _, v := range in {
		if a, ok := v.(slog.Attr); ok {
			out = append(out, a)
		}
	}
	return out
}
