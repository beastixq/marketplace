package web

import (
	"errors"
	"net/http"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/beastixq/marketplace/internal/validators"
)

// --- Profile ---

func (wh *WebHandler) Profile(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	actor := user.actor()
	u, err := wh.userService.GetUserByID(r.Context(), actor, user.UserID)
	if err != nil {
		http.Error(w, "Failed to load profile", http.StatusInternalServerError)
		return
	}

	wh.render(w, "profile", map[string]any{
		"User":    user,
		"Profile": u,
		"Error":   "",
		"Success": "",
	})
}

func (wh *WebHandler) ProfileUpdate(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	fullName := r.FormValue("full_name")
	email := r.FormValue("email")
	phone := r.FormValue("phone")

	update := model.UserUpdate{}
	if fullName != "" {
		if err := validators.ValidateFullName(fullName); err != nil {
			u, _ := wh.userService.GetUserByID(r.Context(), user.actor(), user.UserID)
			wh.render(w, "profile", map[string]any{
				"User":    user,
				"Profile": u,
				"Error":   err.Error(),
				"Success": "",
			})
			return
		}
		update.FullName = &fullName
	}
	if email != "" {
		if err := validators.ValidateEmail(email); err != nil {
			u, _ := wh.userService.GetUserByID(r.Context(), user.actor(), user.UserID)
			wh.render(w, "profile", map[string]any{
				"User":    user,
				"Profile": u,
				"Error":   err.Error(),
				"Success": "",
			})
			return
		}
		update.Email = &email
	}
	if phone != "" {
		if err := validators.ValidatePhone(phone); err != nil {
			u, _ := wh.userService.GetUserByID(r.Context(), user.actor(), user.UserID)
			wh.render(w, "profile", map[string]any{
				"User":    user,
				"Profile": u,
				"Error":   err.Error(),
				"Success": "",
			})
			return
		}
		update.Phone = &phone
	}

	actor := user.actor()
	_, err := wh.userService.UpdateUser(r.Context(), actor, user.UserID, update)

	u, _ := wh.userService.GetUserByID(r.Context(), actor, user.UserID)

	if err != nil {
		errMsg := "Failed to update profile"
		switch {
		case errors.Is(err, service.ErrPhoneAlreadyExists):
			errMsg = service.ErrPhoneAlreadyExists.Error()
		case errors.Is(err, service.ErrEmailAlreadyInUse):
			errMsg = service.ErrEmailAlreadyInUse.Error()
		}
		wh.render(w, "profile", map[string]any{
			"User":    user,
			"Profile": u,
			"Error":   errMsg,
			"Success": "",
		})
		return
	}

	wh.render(w, "profile", map[string]any{
		"User":    user,
		"Profile": u,
		"Error":   "",
		"Success": "Profile updated successfully",
	})
}
