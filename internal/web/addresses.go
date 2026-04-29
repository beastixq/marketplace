package web

import (
	"log"
	"net/http"
	"strconv"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/go-chi/chi/v5"
)

// --- Addresses ---

func (wh *WebHandler) Addresses(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role == "seller" {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), user.actor())

	wh.render(w, "addresses", map[string]any{
		"User":      user,
		"Addresses": addresses,
		"Error":     "",
		"Success":   "",
	})
}

func (wh *WebHandler) AddressCreate(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role == "seller" {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	city := r.FormValue("city")
	street := r.FormValue("street")
	house := r.FormValue("house")
	zipCode := r.FormValue("zip_code")
	isDefault := r.FormValue("is_default") == "on"

	actor := user.actor()
	_, err := wh.addressService.CreateAddress(r.Context(), actor, model.AddressCreate{
		City:      city,
		Street:    street,
		House:     house,
		ZipCode:   zipCode,
		IsDefault: isDefault,
	})

	addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), actor)

	if err != nil {
		wh.render(w, "addresses", map[string]any{
			"User":      user,
			"Addresses": addresses,
			"Error":     "Failed to create address: " + err.Error(),
			"Success":   "",
		})
		return
	}

	wh.render(w, "addresses", map[string]any{
		"User":      user,
		"Addresses": addresses,
		"Error":     "",
		"Success":   "Address added",
	})
}

func (wh *WebHandler) AddressSetDefault(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/addresses", http.StatusSeeOther)
		return
	}

	isDefault := true
	_, err = wh.addressService.UpdateAddress(r.Context(), user.actor(), id, model.AddressUpdate{IsDefault: &isDefault})
	if err != nil {
		log.Printf("AddressSetDefault error: %v", err)
	}
	http.Redirect(w, r, "/addresses", http.StatusSeeOther)
}

func (wh *WebHandler) AddressDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role == "seller" {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/addresses", http.StatusSeeOther)
		return
	}

	_ = wh.addressService.DeleteAddressByID(r.Context(), user.actor(), id)
	http.Redirect(w, r, "/addresses", http.StatusSeeOther)
}
