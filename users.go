package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func user_profile_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := strings.TrimPrefix(r.URL.Path, "/users/")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	intID, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(w, "Invalid User ID", http.StatusBadRequest)
		return
	}

	authenticated, logged_user_id := is_authenticated(r)
	if !authenticated {
		log.Println("User not authenticated, redirecting to login")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	log.Printf("Authenticated user ID: %d, requesting profile for: %d", logged_user_id, intID)

	user_data := get_user_data(intID)
	if user_data == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	is_own_profile := (logged_user_id == intID)
	template_data := struct {
		User         *UserData
		IsOwnProfile bool
		LoggedUserID int
	}{
		User:         user_data,
		IsOwnProfile: is_own_profile,
		LoggedUserID: logged_user_id,
	}

	log.Printf("Rendering profile for user: %s (ID: %d), own profile: %t", user_data.Username, user_data.ID, is_own_profile)

	tmpl := template.Must(template.ParseFiles("template/users_page.html"))
	err = tmpl.Execute(w, template_data)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error executing template:", err)
	}
}

func my_profile_handler(w http.ResponseWriter, r *http.Request) {
	user_id := get_current_user_id(r)
	if user_id == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/users/"+strconv.Itoa(user_id), http.StatusSeeOther)
}
