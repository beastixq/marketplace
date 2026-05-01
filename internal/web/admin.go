package web

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
)

// --- Admin Pages ---

func (wh *WebHandler) AdminUsers(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	const perPage = 50

	search := r.URL.Query().Get("search")
	role := r.URL.Query().Get("role")

	opts := model.UserListOptions{
		Pagination: model.PaginationOpts{Page: page, Limit: perPage},
	}
	if search != "" {
		opts.Search = &search
	}
	if role != "" {
		opts.Role = &role
	}

	users, _ := wh.userService.GetUsers(r.Context(), user.actor(), opts)

	wh.render(w, "admin-users", map[string]any{
		"User":       user,
		"Users":      users,
		"Pagination": map[string]int{"Page": page},
		"HasMore":    len(users) == perPage,
		"Search":     search,
		"Role":       role,
		"Success":    r.URL.Query().Get("success"),
		"Error":      r.URL.Query().Get("error"),
	})
}

func (wh *WebHandler) AdminUserEdit(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	editUser, err := wh.userService.GetUserByID(r.Context(), user.actor(), id)
	if err != nil {
		http.Redirect(w, r, "/admin/users?error=User+not+found", http.StatusSeeOther)
		return
	}

	wh.render(w, "admin-user-edit", map[string]any{
		"User":     user,
		"EditUser": editUser,
		"Success":  r.URL.Query().Get("success"),
		"Error":    r.URL.Query().Get("error"),
	})
}

func (wh *WebHandler) AdminUserEditSubmit(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")
	fullName := r.FormValue("full_name")
	phone := r.FormValue("phone")
	roleStr := r.FormValue("role")
	role := model.UserRole(roleStr)

	uu := model.UserUpdate{
		Email:    &email,
		FullName: &fullName,
		Role:     &role,
	}
	if phone != "" {
		uu.Phone = &phone
	}

	_, err = wh.userService.UpdateUser(r.Context(), user.actor(), id, uu)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%d?error=%s", id, url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/users/%d?success=User+updated", id), http.StatusSeeOther)
}

func (wh *WebHandler) AdminUserDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	if err = wh.userService.DeleteUserByID(r.Context(), user.actor(), id); err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%d?error=%s", id, url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/users?success=User+deleted", http.StatusSeeOther)
}

type adminCategoriesPageData struct {
	User             *userInfo
	Categories       []model.Category
	ParentCategories []model.Category
	Filter           categoryFilter
	HasMore          bool
	Success          string
	Error            string
}

func (d adminCategoriesPageData) PageURL(page int) string {
	return d.Filter.paginationURL("/admin/categories", page)
}

func (wh *WebHandler) AdminCategories(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	filter := parseCategoryFilter(r)
	opts, errorMsg := filter.listOptions(adminCategoryPageSize)
	var categories []model.Category
	if errorMsg == "" {
		var err error
		categories, err = wh.categoryService.GetCategories(r.Context(), opts)
		if err != nil {
			errorMsg = err.Error()
		}
	}
	if queryError := r.URL.Query().Get("error"); queryError != "" {
		errorMsg = queryError
	}
	parentCategories, _ := wh.categoryService.GetCategories(r.Context(), model.CategoryListOptions{
		Pagination: model.PaginationOpts{Page: 1, Limit: categorySelectPageSize},
	})

	wh.render(w, "admin-categories", adminCategoriesPageData{
		User:             user,
		Categories:       categories,
		ParentCategories: parentCategories,
		Filter:           filter,
		HasMore:          len(categories) == adminCategoryPageSize,
		Success:          r.URL.Query().Get("success"),
		Error:            errorMsg,
	})
}

func (wh *WebHandler) AdminCategoryCreate(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	parentIDRaw := r.FormValue("parent_id")

	cc := model.CategoryCreate{Name: name}
	if description != "" {
		cc.Description = &description
	}
	if parentIDRaw != "" {
		parentID, err := strconv.ParseInt(parentIDRaw, 10, 64)
		if err != nil {
			http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape("Invalid parent category"), http.StatusSeeOther)
			return
		}
		cc.ParentID = &parentID
	}

	_, err := wh.categoryService.CreateCategory(r.Context(), user.actor(), cc)
	if err != nil {
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/categories?success=Category+created", http.StatusSeeOther)
}

func (wh *WebHandler) AdminCategoryUpdate(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape("Invalid category id"), http.StatusSeeOther)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape("Category name is required"), http.StatusSeeOther)
		return
	}
	description := r.FormValue("description")

	update := model.CategoryUpdate{
		Name:        &name,
		Description: &description,
	}

	if _, err = wh.categoryService.UpdateCategory(r.Context(), user.actor(), id, update); err != nil {
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/categories?success=Category+updated", http.StatusSeeOther)
}

func (wh *WebHandler) AdminCategoryDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
		return
	}

	if err = wh.categoryService.DeleteCategoryByID(r.Context(), user.actor(), id); err != nil {
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/categories?success=Category+deleted", http.StatusSeeOther)
}

func (wh *WebHandler) AdminOrders(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	const perPage = 50
	statusFilter := r.URL.Query().Get("status")
	status, ok := parseAdminOrderStatus(statusFilter)
	orderIDRaw := r.URL.Query().Get("order_id")

	var orders []model.Order
	var errorMsg string
	if !ok {
		errorMsg = service.ErrOrderStatusInvalid.Error()
	} else if orderIDRaw != "" {
		orderID, err := strconv.ParseInt(orderIDRaw, 10, 64)
		if err != nil || orderID <= 0 {
			errorMsg = "Invalid order id"
		} else {
			order, err := wh.orderService.GetOrderByID(r.Context(), user.actor(), orderID)
			if err != nil {
				errorMsg = err.Error()
			} else if status == nil || order.Status == *status {
				orders = []model.Order{order}
			}
		}
	} else {
		var err error
		orders, err = wh.backofficeService.GetAdminOrders(r.Context(), user.actor(), model.AdminOrderListOptions{
			Status: status,
			Pagination: model.PaginationOpts{
				Page:  page,
				Limit: perPage,
			},
		})
		if err != nil {
			errorMsg = err.Error()
		}
	}

	wh.render(w, "admin-orders", map[string]any{
		"User":    user,
		"Orders":  orders,
		"Page":    page,
		"HasMore": orderIDRaw == "" && len(orders) == perPage,
		"Status":  statusFilter,
		"OrderID": orderIDRaw,
		"Success": r.URL.Query().Get("success"),
		"Error":   errorMsg,
	})
}

func parseAdminOrderStatus(raw string) (*model.OrderStatus, bool) {
	if raw == "" {
		return nil, true
	}
	status := model.OrderStatus(raw)
	switch status {
	case model.StatusPending, model.StatusPaid, model.StatusShipped, model.StatusDelivered, model.StatusCancelled:
		return &status, true
	default:
		return nil, false
	}
}

// Admin moderation: delete product (bypasses ownership check)
func (wh *WebHandler) AdminProductDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err = wh.productService.DeleteProductByID(r.Context(), user.actor(), id); err != nil {
		log.Printf("AdminProductDelete error: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/products/%d", id), http.StatusSeeOther)
}

func (wh *WebHandler) AdminProductCategoriesUpdate(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/?error="+url.QueryEscape("Invalid product id"), http.StatusSeeOther)
		return
	}
	categoryIDs, categoryErr := parseRequiredCategoryIDs(r)
	if categoryErr != "" {
		http.Redirect(w, r, fmt.Sprintf("/products/%d?category_error=%s", id, url.QueryEscape(categoryErr)), http.StatusSeeOther)
		return
	}

	if err = wh.productService.ReplaceProductCategories(r.Context(), user.actor(), id, categoryIDs); err != nil {
		http.Redirect(w, r, fmt.Sprintf("/products/%d?category_error=%s", id, url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/products/%d?notice=categories-updated", id), http.StatusSeeOther)
}

// Admin moderation: delete seller
func (wh *WebHandler) AdminSellerDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	if err = wh.sellerService.DeleteSellerByID(r.Context(), user.actor(), id); err != nil {
		log.Printf("AdminSellerDelete error: %v", err)
	}
	http.Redirect(w, r, "/admin/users?success=Seller+deleted", http.StatusSeeOther)
}

// Admin moderation: delete review
func (wh *WebHandler) AdminReviewDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get product ID for redirect
	var productID int64
	review, err := wh.reviewService.GetReviewByID(r.Context(), id)
	if err == nil {
		productID = review.ProductID
	}

	if err = wh.reviewService.DeleteReviewByID(r.Context(), user.actor(), id); err != nil {
		log.Printf("AdminReviewDelete error: %v", err)
	}

	if productID > 0 {
		http.Redirect(w, r, fmt.Sprintf("/products/%d", productID), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
