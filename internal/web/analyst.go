package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/beastixq/marketplace/internal/model"
)

// --- Analyst Dashboard ---

type analystReportFilter struct {
	DateFrom string
	DateTo   string
	Period   string
	Limit    string
}

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

	filter, reportOpts, errorMsg := parseAnalystReportOptions(r)
	var orderDynamics []model.OrderDynamicsPoint
	var salesByCategory []model.CategorySalesStats
	if errorMsg == "" {
		orderDynamics, err = wh.backofficeService.GetOrderDynamics(r.Context(), user.actor(), reportOpts)
		if err != nil {
			errorMsg = err.Error()
		}
	}
	if errorMsg == "" {
		salesByCategory, err = wh.backofficeService.GetSalesByCategory(r.Context(), user.actor(), reportOpts)
		if err != nil {
			errorMsg = err.Error()
		}
	}

	wh.render(w, "analyst", map[string]any{
		"User":            user,
		"Stats":           stats,
		"Filter":          filter,
		"OrderDynamics":   orderDynamics,
		"SalesByCategory": salesByCategory,
		"Error":           errorMsg,
	})
}

func parseAnalystReportOptions(r *http.Request) (analystReportFilter, model.ReportOptions, string) {
	q := r.URL.Query()
	filter := analystReportFilter{
		DateFrom: q.Get("date_from"),
		DateTo:   q.Get("date_to"),
		Period:   q.Get("period"),
		Limit:    q.Get("limit"),
	}
	if filter.Period == "" {
		filter.Period = string(model.ReportPeriodDay)
	}
	if filter.Limit == "" {
		filter.Limit = "10"
	}

	opts := model.ReportOptions{
		Period: model.ReportPeriod(filter.Period),
	}

	if filter.DateFrom != "" {
		dateFrom, err := time.ParseInLocation("2006-01-02", filter.DateFrom, time.Local)
		if err != nil {
			return filter, model.ReportOptions{}, "Invalid start date"
		}
		opts.DateFrom = &dateFrom
	}
	if filter.DateTo != "" {
		dateTo, err := time.ParseInLocation("2006-01-02", filter.DateTo, time.Local)
		if err != nil {
			return filter, model.ReportOptions{}, "Invalid end date"
		}
		dateTo = dateTo.AddDate(0, 0, 1).Add(-time.Nanosecond)
		opts.DateTo = &dateTo
	}
	if filter.Limit != "" {
		limit, err := strconv.Atoi(filter.Limit)
		if err != nil || limit < 1 || limit > 50 {
			return filter, model.ReportOptions{}, "Category limit must be between 1 and 50"
		}
		opts.Limit = limit
	}

	return filter, opts, ""
}
