package web

import (
	"net/http"
)

// --- Analyst Dashboard ---

func (wh *WebHandler) AnalystDashboard(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "analyst", "admin")
	if user == nil {
		return
	}

	stats, err := wh.backofficeService.GetPlatformStats(r.Context(), user.actor())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wh.render(w, "analyst", map[string]any{
		"User":  user,
		"Stats": stats,
	})
}
