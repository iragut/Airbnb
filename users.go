package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func get_user_listings(userID int, isAdmin bool) []Listing {
	var listings []Listing
	var query string
	var args []interface{}

	if isAdmin {
		// If user is admin, show all listings
		query = `
			SELECT p.id, p.user_id, p.title, p.country, p.city, p.address, 
			       p.description, p.price, p.type, p.created_at,
			       COALESCE(MIN(i.image_url), '') as image_url,
			       COALESCE(MAX(a.wifi), false) as has_wifi,
			       COALESCE(MAX(a.kitchen), false) as has_kitchen,
			       COALESCE(MAX(a.air_conditioning), false) as has_ac,
			       COALESCE(MAX(a.parking), false) as has_parking
			FROM Posts p
			LEFT JOIN Images i ON p.id = i.post_id
			LEFT JOIN Amenities a ON p.id = a.post_id
			GROUP BY p.id, p.user_id, p.title, p.country, p.city, p.address, p.description, p.price, p.type, p.created_at
			ORDER BY p.created_at DESC`
		args = []interface{}{}
	} else {
		// For regular users, show only their listings
		query = `
			SELECT p.id, p.user_id, p.title, p.country, p.city, p.address, 
			       p.description, p.price, p.type, p.created_at,
			       COALESCE(MIN(i.image_url), '') as image_url,
			       COALESCE(MAX(a.wifi), false) as has_wifi,
			       COALESCE(MAX(a.kitchen), false) as has_kitchen,
			       COALESCE(MAX(a.air_conditioning), false) as has_ac,
			       COALESCE(MAX(a.parking), false) as has_parking
			FROM Posts p
			LEFT JOIN Images i ON p.id = i.post_id
			LEFT JOIN Amenities a ON p.id = a.post_id
			WHERE p.user_id = ?
			GROUP BY p.id, p.user_id, p.title, p.country, p.city, p.address, p.description, p.price, p.type, p.created_at
			ORDER BY p.created_at DESC`
		args = []interface{}{userID}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying user listings: %v", err)
		return listings
	}
	defer rows.Close()

	for rows.Next() {
		var listing Listing
		err := rows.Scan(
			&listing.ID, &listing.UserID, &listing.Title, &listing.Country,
			&listing.City, &listing.Address, &listing.Description,
			&listing.Price, &listing.Type, &listing.CreatedAt,
			&listing.ImageURL, &listing.HasWifi, &listing.HasKitchen,
			&listing.HasAC, &listing.HasParking,
		)
		if err != nil {
			log.Printf("Error scanning listing: %v", err)
			continue
		}
		listings = append(listings, listing)
	}

	return listings
}

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

	// Get user's listings (or all listings if admin)
	isAdmin := user_data.Role == "admin" || user_data.Role == "moderator"
	userListings := get_user_listings(intID, isAdmin)

	template_data := struct {
		User         *UserData
		IsOwnProfile bool
		LoggedUserID int
		Listings     []Listing
		IsAdmin      bool
	}{
		User:         user_data,
		IsOwnProfile: is_own_profile,
		LoggedUserID: logged_user_id,
		Listings:     userListings,
		IsAdmin:      isAdmin,
	}

	log.Printf("Rendering profile for user: %s (ID: %d), own profile: %t, listings count: %d",
		user_data.Username, user_data.ID, is_own_profile, len(userListings))

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
