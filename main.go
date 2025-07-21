package main

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/sessions"
)

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

func is_authenticated(r *http.Request) (bool, int) {
	session, _ := store.Get(r, "cookie-name")

	auth, ok := session.Values["authenticated"].(bool)
	if !ok || !auth {
		return false, 0
	}

	user_id, ok := session.Values["user_id"].(int)
	if !ok {
		return false, 0
	}

	return true, user_id
}

func main_page_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tmpl := template.Must(template.ParseFiles("template/main_page.html"))
	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error executing template:", err)
	}
}

func login_handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("template/login_page.html"))
		err := tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("Error executing template:", err)
		}
	case http.MethodPost:
		session, _ := store.Get(r, "cookie-name")
		email := r.FormValue("email")
		password := r.FormValue("password")

		log.Println("Login attempt:", email)

		if check_user_exists(email, password) {
			log.Println("User authenticated:", email)

			user_id := get_user_id(email)
			if user_id == 0 {
				http.Error(w, "User not found", http.StatusInternalServerError)
				return
			}

			session.Values["authenticated"] = true
			session.Values["user_id"] = user_id
			session.Values["email"] = email

			err := session.Save(r, w)
			if err != nil {
				log.Println("Error saving session:", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			log.Println("Redirect to user profile:", user_id)
			http.Redirect(w, r, "/users/"+strconv.Itoa(user_id), http.StatusSeeOther)
		} else {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func logout_handler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cookie-name")

	// Clear session values
	session.Values["authenticated"] = false
	delete(session.Values, "user_id")
	delete(session.Values, "email")

	// Set MaxAge to -1 to delete the cookie
	session.Options.MaxAge = -1

	err := session.Save(r, w)
	if err != nil {
		log.Println("Error clearing session:", err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func register_handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("template/register_page.html"))
		err := tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("Error executing template:", err)
		}
	case http.MethodPost:
		email := r.FormValue("email")
		password := r.FormValue("password")
		tel_number := r.FormValue("phone_number")
		create_user(email, password, "New User", tel_number, "user")
		log.Println("User registered:", email)

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

}

func explore_handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get post counts for each city
		number_of_posts_city := make(map[string]int)

		number_of_posts_city["New York"] = get_post_count_by_city("New York")
		number_of_posts_city["Paris"] = get_post_count_by_city("Paris")
		number_of_posts_city["Tokyo"] = get_post_count_by_city("Tokyo")
		number_of_posts_city["London"] = get_post_count_by_city("London")
		number_of_posts_city["Bali"] = get_post_count_by_city("Bali")
		number_of_posts_city["Dubai"] = get_post_count_by_city("Dubai")
		number_of_posts_city["Rome"] = get_post_count_by_city("Rome")
		number_of_posts_city["Barcelona"] = get_post_count_by_city("Barcelona")

		template_data := struct {
			CityAndPosts map[string]int
		}{
			CityAndPosts: number_of_posts_city,
		}

		tmpl := template.Must(template.ParseFiles("template/explore_page.html"))
		err := tmpl.Execute(w, template_data)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("Error executing template:", err)
		}

	case http.MethodPost:
		destination := r.FormValue("destination")
		checkin := r.FormValue("checkin")
		checkout := r.FormValue("checkout")
		guests := r.FormValue("guests")

		log.Printf("Search request: destination=%s, checkin=%s, checkout=%s, guests=%s",
			destination, checkin, checkout, guests)

		// Redirect to listings page with search parameters
		redirectURL := "/listings"
		if destination != "" {
			redirectURL += "?destination=" + destination
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func buildPaginationQuery(values url.Values) string {
	// Remove page parameter and build query string for pagination
	newValues := url.Values{}
	for key, vals := range values {
		if key != "page" {
			for _, val := range vals {
				if val != "" {
					newValues.Add(key, val)
				}
			}
		}
	}
	return newValues.Encode()
}

func listings_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse search parameters
	params := SearchParams{
		Destination:  strings.TrimSpace(r.URL.Query().Get("destination")),
		CheckIn:      r.URL.Query().Get("checkin"),
		CheckOut:     r.URL.Query().Get("checkout"),
		Guests:       r.URL.Query().Get("guests"),
		PropertyType: r.URL.Query().Get("type"),
		Page:         1,
		Limit:        20,
	}

	// Parse page number
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}

	// Parse price filters
	if minPriceStr := r.URL.Query().Get("min_price"); minPriceStr != "" {
		if minPrice, err := strconv.ParseFloat(minPriceStr, 64); err == nil && minPrice >= 0 {
			params.MinPrice = minPrice
		}
	}

	if maxPriceStr := r.URL.Query().Get("max_price"); maxPriceStr != "" {
		if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil && maxPrice > 0 {
			params.MaxPrice = maxPrice
		}
	}

	// Search listings
	result, err := search_listings(params)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error searching listings:", err)
		return
	}

	// Prepare pagination data
	pageNumbers := make([]int, 0)
	start := max(1, params.Page-2)
	end := min(result.TotalPages, params.Page+2)

	for i := start; i <= end; i++ {
		pageNumbers = append(pageNumbers, i)
	}

	// Build pagination query string (without page parameter)
	paginationQuery := buildPaginationQuery(r.URL.Query())

	// Prepare template data
	templateData := struct {
		Listings        []Listing
		SearchParams    SearchParams
		SearchQuery     string
		TotalResults    int
		TotalPages      int
		CurrentPage     int
		NextPage        int
		PrevPage        int
		PageNumbers     []int
		PaginationQuery string
	}{
		Listings:        result.Listings,
		SearchParams:    params,
		SearchQuery:     params.Destination,
		TotalResults:    result.TotalResults,
		TotalPages:      result.TotalPages,
		CurrentPage:     result.CurrentPage,
		NextPage:        result.CurrentPage + 1,
		PrevPage:        result.CurrentPage - 1,
		PageNumbers:     pageNumbers,
		PaginationQuery: paginationQuery,
	}

	tmpl := template.Must(template.ParseFiles("template/listings_page.html"))
	err = tmpl.Execute(w, templateData)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error executing template:", err)
	}
}

func main() {
	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", main_page_handler)
	http.HandleFunc("/login", login_handler)
	http.HandleFunc("/logout", logout_handler)

	http.HandleFunc("/register", register_handler)
	http.HandleFunc("/explore", explore_handler)
	http.HandleFunc("/listings", listings_handler)

	http.HandleFunc("/users/", user_profile_handler)
	http.HandleFunc("/my-profile", my_profile_handler)

	log.Println("Server starting on :8080")

	init_database()
	http.ListenAndServe(":8080", nil)
}
