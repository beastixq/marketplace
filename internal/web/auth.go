package web

import (
	"net/http"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/validators"
)

// --- Auth ---

func (wh *WebHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	wh.render(w, "login", map[string]any{
		"Email": "",
		"Error": "",
		"User":  wh.userFromCookie(r),
	})
}

func (wh *WebHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	token, err := wh.authService.Login(r.Context(), email, password)
	if err != nil {
		wh.render(w, "login", map[string]any{
			"Email": email,
			"Error": "Invalid email or password",
			"User":  nil,
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (wh *WebHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	wh.render(w, "register", map[string]any{
		"Email":    "",
		"FullName": "",
		"Phone":    "",
		"Role":     "buyer",
		"Error":    "",
		"User":     wh.userFromCookie(r),
	})
}

func (wh *WebHandler) RegisterSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	fullName := r.FormValue("full_name")
	password := r.FormValue("password")
	phone := r.FormValue("phone")
	role := r.FormValue("role")

	renderErr := func(msg string) {
		wh.render(w, "register", map[string]any{
			"Email":    email,
			"FullName": fullName,
			"Phone":    phone,
			"Role":     role,
			"Error":    msg,
			"User":     nil,
		})
	}

	if err := validators.ValidateEmail(email); err != nil {
		renderErr(err.Error())
		return
	}
	if err := validators.ValidateFullName(fullName); err != nil {
		renderErr(err.Error())
		return
	}
	if err := validators.ValidatePassword(password); err != nil {
		renderErr(err.Error())
		return
	}
	if phone != "" {
		if err := validators.ValidatePhone(phone); err != nil {
			renderErr(err.Error())
			return
		}
	}

	uc := model.UserCreate{
		Email:    email,
		FullName: fullName,
		Password: password,
		Role:     model.UserRole(role),
	}
	if phone != "" {
		uc.Phone = &phone
	}

	token, err := wh.authService.Register(r.Context(), uc)
	if err != nil {
		wh.render(w, "register", map[string]any{
			"Email":    email,
			"FullName": fullName,
			"Phone":    phone,
			"Role":     role,
			"Error":    "Registration failed: " + err.Error(),
			"User":     nil,
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (wh *WebHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
