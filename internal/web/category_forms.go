package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/beastixq/marketplace/internal/model"
)

const (
	publicCategoryPageSize   = 24
	adminCategoryPageSize    = 50
	categorySelectPageSize   = 500
	categoryPickerPageSize   = 8
	categoryParentFilterRoot = "root"
)

type categoryFilter struct {
	Search string
	Parent string
	Page   int
}

func (f categoryFilter) listOptions(limit int) (model.CategoryListOptions, string) {
	opts := model.CategoryListOptions{
		Pagination: model.PaginationOpts{Page: f.Page, Limit: limit},
	}
	if f.Search != "" {
		opts.Search = &f.Search
	}
	if f.Parent == categoryParentFilterRoot {
		opts.OnlyRoot = true
		return opts, ""
	}
	if f.Parent != "" {
		parentID, err := strconv.ParseInt(f.Parent, 10, 64)
		if err != nil {
			return opts, "Invalid parent category"
		}
		opts.ParentID = &parentID
	}
	return opts, ""
}

func (f categoryFilter) paginationURL(path string, page int) string {
	v := url.Values{}
	if f.Search != "" {
		v.Set("search", f.Search)
	}
	if f.Parent != "" {
		v.Set("parent", f.Parent)
	}
	v.Set("page", strconv.Itoa(page))
	return path + "?" + v.Encode()
}

func parseCategoryFilter(r *http.Request) categoryFilter {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	return categoryFilter{
		Search: r.URL.Query().Get("search"),
		Parent: r.URL.Query().Get("parent"),
		Page:   page,
	}
}

type categoryPickerData struct {
	Search       string
	Page         int
	HasMore      bool
	Options      []model.Category
	Selected     []model.Category
	PrevURL      string
	NextURL      string
	SearchAction string
}

func (wh *WebHandler) categoryPickerData(r *http.Request, actionPath string, selected []model.Category) categoryPickerData {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("category_page"))
	if page < 1 {
		page = 1
	}
	search := q.Get("category_search")

	opts := model.CategoryListOptions{
		Pagination: model.PaginationOpts{Page: page, Limit: categoryPickerPageSize},
	}
	if search != "" {
		opts.Search = &search
	}

	options, _ := wh.categoryService.GetCategories(r.Context(), opts)
	selectedIDs := selectedCategoryMap(selected)
	filtered := make([]model.Category, 0, len(options))
	for _, option := range options {
		if selectedIDs[option.ID] {
			continue
		}
		filtered = append(filtered, option)
	}

	return categoryPickerData{
		Search:       search,
		Page:         page,
		HasMore:      len(options) == categoryPickerPageSize,
		Options:      filtered,
		Selected:     selected,
		PrevURL:      categoryPickerURL(r, actionPath, page-1),
		NextURL:      categoryPickerURL(r, actionPath, page+1),
		SearchAction: actionPath,
	}
}

func categoryPickerURL(r *http.Request, path string, page int) string {
	v := url.Values{}
	for key, values := range r.URL.Query() {
		for _, value := range values {
			v.Add(key, value)
		}
	}
	v.Set("category_page", strconv.Itoa(page))
	return path + "?" + v.Encode()
}

func parseRequiredCategoryIDs(r *http.Request) ([]int64, string) {
	if err := r.ParseForm(); err != nil {
		return nil, "Invalid category form"
	}
	rawIDs := r.Form["category_ids"]
	if len(rawIDs) == 0 {
		return nil, "Choose at least one category"
	}

	seen := make(map[int64]struct{}, len(rawIDs))
	ids := make([]int64, 0, len(rawIDs))
	for _, rawID := range rawIDs {
		id, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Sprintf("Invalid category id: %s", rawID)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, "Choose at least one category"
	}
	return ids, ""
}

func selectedCategoryMap(categories []model.Category) map[int64]bool {
	selected := make(map[int64]bool, len(categories))
	for _, category := range categories {
		selected[category.ID] = true
	}
	return selected
}
