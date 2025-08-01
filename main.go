package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/nfnt/resize"
)

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key   = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

// AuthContext holds authentication information for templates
type AuthContext struct {
	IsAuthenticated bool
	UserID          int
	Username        string
}

type PersonalData struct {
	UserID    int
	FirstName string
	LastName  string
	BirthDate string
}

// Image struct for listing images
type PropertyImage struct {
	ID       int
	PostID   int
	ImageURL string
}

func edit_profile_handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		authenticated, userID := is_authenticated(r)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		authCtx := get_auth(r)
		userData := get_user_data(userID)
		if userData == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Get personal data
		personalData := get_personal_data(userID)

		// Parse phone number to get country code and number
		countryCode, phoneNumber := parse_phone_number(userData.PhoneNumber)

		templateData := struct {
			Auth         AuthContext
			User         *UserData
			PersonalData *PersonalData
			CountryCode  string
			PhoneNumber  string
		}{
			Auth:         authCtx,
			User:         userData,
			PersonalData: personalData,
			CountryCode:  countryCode,
			PhoneNumber:  phoneNumber,
		}

		tmpl := template.Must(template.ParseFiles("template/edit_profile.html"))
		err := tmpl.Execute(w, templateData)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("Error executing template:", err)
		}

	case http.MethodPost:
		authenticated, userID := is_authenticated(r)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Parse form data
		firstName := strings.TrimSpace(r.FormValue("first_name"))
		lastName := strings.TrimSpace(r.FormValue("last_name"))
		username := strings.TrimSpace(r.FormValue("username"))
		email := strings.TrimSpace(r.FormValue("email"))
		birthDate := r.FormValue("birth_date")
		countryCode := r.FormValue("country_code")
		phoneNumber := r.FormValue("number")

		// Password fields
		currentPassword := r.FormValue("current_password")
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_password")

		// Validate required fields
		if username == "" || email == "" || phoneNumber == "" {
			http.Error(w, "Username, email, and phone number are required", http.StatusBadRequest)
			return
		}

		// Validate password change if provided
		if currentPassword != "" || newPassword != "" || confirmPassword != "" {
			if currentPassword == "" {
				http.Error(w, "Current password is required to change password", http.StatusBadRequest)
				return
			}

			// Verify current password
			if !verify_current_password(userID, currentPassword) {
				http.Error(w, "Current password is incorrect", http.StatusBadRequest)
				return
			}

			if newPassword != confirmPassword {
				http.Error(w, "New passwords do not match", http.StatusBadRequest)
				return
			}

			if len(newPassword) < 8 {
				http.Error(w, "New password must be at least 8 characters long", http.StatusBadRequest)
				return
			}

			// Update password
			err := update_user_password(userID, newPassword)
			if err != nil {
				http.Error(w, "Error updating password", http.StatusInternalServerError)
				return
			}
		}

		// Update user data
		fullPhoneNumber := countryCode + phoneNumber
		err := update_user_profile(userID, username, email, fullPhoneNumber)
		if err != nil {
			http.Error(w, "Error updating profile: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update personal data
		err = update_personal_data(userID, firstName, lastName, birthDate)
		if err != nil {
			log.Printf("Error updating personal data: %v", err)
		}

		// Redirect back to profile
		http.Redirect(w, r, "/my-profile", http.StatusSeeOther)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func edit_listing_handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		authenticated, userID := is_authenticated(r)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract listing ID from URL
		path := strings.TrimPrefix(r.URL.Path, "/edit-listing/")
		if path == "" {
			http.Error(w, "Listing ID is required", http.StatusBadRequest)
			return
		}

		listingID, err := strconv.Atoi(path)
		if err != nil {
			http.Error(w, "Invalid listing ID", http.StatusBadRequest)
			return
		}

		// Get listing details
		listing, err := get_listing_by_id(listingID)
		if err != nil || listing == nil {
			http.Error(w, "Listing not found", http.StatusNotFound)
			return
		}

		// Check if user owns this listing
		if listing.UserID != userID {
			http.Error(w, "You don't own this listing", http.StatusForbidden)
			return
		}

		// Get amenities
		amenities := get_listing_amenities(listingID)

		// Get images
		images := get_listing_images(listingID)

		authCtx := get_auth(r)

		templateData := struct {
			Auth      AuthContext
			Listing   *Listing
			Amenities *PropertyAmenities
			Images    []PropertyImage
		}{
			Auth:      authCtx,
			Listing:   listing,
			Amenities: amenities,
			Images:    images,
		}

		tmpl := template.Must(template.ParseFiles("template/edit_listing.html"))
		err = tmpl.Execute(w, templateData)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("Error executing template:", err)
		}

	case http.MethodPost:
		authenticated, userID := is_authenticated(r)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract listing ID from URL
		path := strings.TrimPrefix(r.URL.Path, "/edit-listing/")
		listingID, err := strconv.Atoi(path)
		if err != nil {
			http.Error(w, "Invalid listing ID", http.StatusBadRequest)
			return
		}

		// Verify ownership
		listing, err := get_listing_by_id(listingID)
		if err != nil || listing == nil || listing.UserID != userID {
			http.Error(w, "Listing not found or access denied", http.StatusForbidden)
			return
		}

		// Parse multipart form for file uploads
		err = r.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		// Handle image deletions
		deleteImages := r.Form["delete_images"]
		for _, imageIDStr := range deleteImages {
			imageID, err := strconv.Atoi(imageIDStr)
			if err == nil {
				delete_listing_image(imageID, listingID)
			}
		}

		// Handle new image uploads
		files := r.MultipartForm.File["images"]
		for _, fileHeader := range files {
			err := save_uploaded_image(fileHeader, listingID)
			if err != nil {
				log.Printf("Error uploading image: %v", err)
				// Continue with other images even if one fails
			}
		}

		// Update listing details
		title := strings.TrimSpace(r.FormValue("title"))
		description := strings.TrimSpace(r.FormValue("description"))
		country := strings.TrimSpace(r.FormValue("country"))
		city := strings.TrimSpace(r.FormValue("city"))
		address := strings.TrimSpace(r.FormValue("address"))
		priceStr := r.FormValue("price")
		propertyType := r.FormValue("type")

		// Validate input
		if title == "" || description == "" || country == "" || city == "" || address == "" || priceStr == "" || propertyType == "" {
			http.Error(w, "All required fields must be filled", http.StatusBadRequest)
			return
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price <= 0 {
			http.Error(w, "Invalid price", http.StatusBadRequest)
			return
		}

		// Update listing
		err = update_listing(listingID, title, country, city, address, description, price, propertyType)
		if err != nil {
			http.Error(w, "Error updating listing: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update amenities
		wifi := r.FormValue("wifi") == "on"
		kitchen := r.FormValue("kitchen") == "on"
		ac := r.FormValue("air_conditioning") == "on"
		parking := r.FormValue("parking") == "on"
		pool := r.FormValue("pool") == "on"
		washer := r.FormValue("washer") == "on"
		dryer := r.FormValue("dryer") == "on"
		tv := r.FormValue("tv") == "on"
		heating := r.FormValue("heating") == "on"
		balcony := r.FormValue("balcony") == "on"
		pets := r.FormValue("pets_allowed") == "on"

		err = update_listing_amenities(listingID, wifi, ac, kitchen, parking, pets, pool, washer, dryer, tv, heating, balcony)
		if err != nil {
			log.Printf("Error updating amenities: %v", err)
		}

		// Redirect to the listing
		http.Redirect(w, r, "/property/"+strconv.Itoa(listingID), http.StatusSeeOther)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func delete_listing_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authenticated, userID := is_authenticated(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Extract listing ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/delete-listing/")
	listingID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid listing ID", http.StatusBadRequest)
		return
	}

	// Verify ownership and delete
	err = delete_listing_by_owner(listingID, userID)
	if err != nil {
		http.Error(w, "Error deleting listing: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to profile
	http.Redirect(w, r, "/my-profile", http.StatusSeeOther)
}

func save_uploaded_image(fileHeader *multipart.FileHeader, listingID int) error {
	// Open uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	// Validate file type
	if !isValidImageType(fileHeader.Header.Get("Content-Type")) {
		return fmt.Errorf("invalid image type")
	}

	// Generate unique filename
	filename := generate_unique_filename(fileHeader.Filename)

	// Create uploads directory if it doesn't exist
	uploadsDir := "static/uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return err
	}

	// Full path for the file
	filePath := filepath.Join(uploadsDir, filename)

	// Create the file
	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Decode and resize image
	img, format, err := image.Decode(file)
	if err != nil {
		return err
	}

	// Resize image to max 1200px width while maintaining aspect ratio
	resized := resize.Resize(1200, 0, img, resize.Lanczos3)

	// Encode and save
	switch format {
	case "jpeg":
		err = jpeg.Encode(dst, resized, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(dst, resized)
	default:
		// Default to JPEG
		err = jpeg.Encode(dst, resized, &jpeg.Options{Quality: 85})
		filename = strings.TrimSuffix(filename, filepath.Ext(filename)) + ".jpg"
		filePath = filepath.Join(uploadsDir, filename)
	}

	if err != nil {
		os.Remove(filePath) // Clean up on error
		return err
	}

	// Save to database
	imageURL := "/static/uploads/" + filename
	return save_listing_image(listingID, imageURL)
}

func isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/webp",
		"image/gif",
	}

	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}

func generate_unique_filename(originalName string) string {
	// Generate random bytes
	bytes := make([]byte, 16)
	rand.Read(bytes)

	// Create unique filename with timestamp and random string
	ext := filepath.Ext(originalName)
	timestamp := time.Now().Unix()
	randomStr := hex.EncodeToString(bytes)

	return fmt.Sprintf("%d_%s%s", timestamp, randomStr, ext)
}

func get_auth(r *http.Request) AuthContext {
	authenticated, userID := is_authenticated(r)
	var username string

	if authenticated {
		if userData := get_user_data(userID); userData != nil {
			username = userData.Username
		}
	}

	return AuthContext{
		IsAuthenticated: authenticated,
		UserID:          userID,
		Username:        username,
	}
}

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

	authCtx := get_auth(r)

	templateData := struct {
		Auth AuthContext
	}{
		Auth: authCtx,
	}

	tmpl := template.Must(template.ParseFiles("template/main_page.html"))
	err := tmpl.Execute(w, templateData)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error executing template:", err)
	}
}

func login_handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		authCtx := get_auth(r)

		// If already logged in, redirect to profile
		if authCtx.IsAuthenticated {
			http.Redirect(w, r, "/my-profile", http.StatusSeeOther)
			return
		}

		templateData := struct {
			Auth AuthContext
		}{
			Auth: authCtx,
		}

		tmpl := template.Must(template.ParseFiles("template/login_page.html"))
		err := tmpl.Execute(w, templateData)
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
		authCtx := get_auth(r)

		// If already logged in, redirect to profile
		if authCtx.IsAuthenticated {
			http.Redirect(w, r, "/my-profile", http.StatusSeeOther)
			return
		}

		templateData := struct {
			Auth AuthContext
		}{
			Auth: authCtx,
		}

		tmpl := template.Must(template.ParseFiles("template/register_page.html"))
		err := tmpl.Execute(w, templateData)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("Error executing template:", err)
		}
	case http.MethodPost:
		email := r.FormValue("email")
		password := r.FormValue("password")
		confirm_password := r.FormValue("confirm_password")
		country_code := r.FormValue("country_code")
		number := r.FormValue("number")

		// Basic validation
		if email == "" || password == "" || number == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		if password != confirm_password {
			http.Error(w, "Passwords do not match", http.StatusBadRequest)
			return
		}

		// Combine country code and number
		phone_number := country_code + number

		create_user(email, password, "New User", phone_number, "user")
		log.Println("User registered:", email)

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func explore_handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		authCtx := get_auth(r)

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
			Auth         AuthContext
		}{
			CityAndPosts: number_of_posts_city,
			Auth:         authCtx,
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

	authCtx := get_auth(r)

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

	// Parse amenity filters
	params.Wifi = r.URL.Query().Get("wifi") == "true"
	params.Kitchen = r.URL.Query().Get("kitchen") == "true"
	params.AirConditioning = r.URL.Query().Get("air_conditioning") == "true"
	params.Parking = r.URL.Query().Get("parking") == "true"
	params.Pool = r.URL.Query().Get("pool") == "true"
	params.TV = r.URL.Query().Get("tv") == "true"
	params.Washer = r.URL.Query().Get("washer") == "true"
	params.Dryer = r.URL.Query().Get("dryer") == "true"
	params.Heating = r.URL.Query().Get("heating") == "true"
	params.Balcony = r.URL.Query().Get("balcony") == "true"
	params.PetsAllowed = r.URL.Query().Get("pets_allowed") == "true"

	// Search listings using enhanced function
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
		Auth            AuthContext
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
		Auth:            authCtx,
	}

	tmpl := template.Must(template.ParseFiles("template/listings_page.html"))
	err = tmpl.Execute(w, templateData)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error executing template:", err)
	}
}

func property_detail_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx := get_auth(r)

	// Extract property ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/property/")
	if path == "" {
		http.Error(w, "Property ID is required", http.StatusBadRequest)
		return
	}

	propertyID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid property ID", http.StatusBadRequest)
		return
	}

	// Get property details
	propertyDetail, err := get_property_detail(propertyID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error fetching property detail:", err)
		return
	}

	if propertyDetail == nil || propertyDetail.Property == nil {
		http.Error(w, "Property not found", http.StatusNotFound)
		return
	}

	// Create template functions
	funcMap := template.FuncMap{
		"title": strings.Title,
		"iterate": func(count int) []int {
			var result []int
			for i := 0; i < count; i++ {
				result = append(result, i)
			}
			return result
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"add": func(a, b, c float64) float64 {
			return a + b + c
		},
	}

	// Prepare template data
	templateData := struct {
		Property  *Listing
		Host      *UserData
		Amenities *PropertyAmenities
		Reviews   []Review
		Auth      AuthContext
	}{
		Property:  propertyDetail.Property,
		Host:      propertyDetail.Host,
		Amenities: propertyDetail.Amenities,
		Reviews:   propertyDetail.Reviews,
		Auth:      authCtx,
	}

	// Parse and execute template
	tmpl := template.Must(template.New("property_detail.html").Funcs(funcMap).ParseFiles("template/property_detail.html"))
	err = tmpl.Execute(w, templateData)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error executing template:", err)
	}
}

func booking_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if user is authenticated
	authenticated, userID := is_authenticated(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form data
	propertyIDStr := r.FormValue("property_id")
	checkin := r.FormValue("checkin")
	checkout := r.FormValue("checkout")
	guestsStr := r.FormValue("guests")

	// Validate input
	if propertyIDStr == "" || checkin == "" || checkout == "" || guestsStr == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	propertyID, err := strconv.Atoi(propertyIDStr)
	if err != nil {
		http.Error(w, "Invalid property ID", http.StatusBadRequest)
		return
	}

	guests, err := strconv.Atoi(guestsStr)
	if err != nil || guests <= 0 {
		http.Error(w, "Invalid number of guests", http.StatusBadRequest)
		return
	}

	// Get property details to find the host
	property, err := get_listing_by_id(propertyID)
	if err != nil || property == nil {
		http.Error(w, "Property not found", http.StatusNotFound)
		return
	}

	// Check if user is trying to book their own property
	if property.UserID == userID {
		http.Error(w, "You cannot book your own property", http.StatusBadRequest)
		return
	}

	// Calculate total price
	totalPrice, nights, err := calculate_booking_price(propertyID, checkin, checkout)
	if err != nil {
		http.Error(w, "Error calculating price: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create booking
	err = create_booking(propertyID, userID, property.UserID, guests, checkin, checkout, totalPrice)
	if err != nil {
		if err.Error() == "dates are not available" {
			http.Error(w, "Sorry, these dates are not available", http.StatusConflict)
			return
		}
		http.Error(w, "Error creating booking: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to booking confirmation page
	http.Redirect(w, r, "/booking-success?property="+propertyIDStr+"&nights="+strconv.Itoa(nights)+"&total="+fmt.Sprintf("%.2f", totalPrice), http.StatusSeeOther)
}

func booking_success_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if user is authenticated
	authenticated, _ := is_authenticated(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	authCtx := get_auth(r)

	// Get query parameters
	propertyIDStr := r.URL.Query().Get("property")
	nightsStr := r.URL.Query().Get("nights")
	totalStr := r.URL.Query().Get("total")

	propertyID, _ := strconv.Atoi(propertyIDStr)
	nights, _ := strconv.Atoi(nightsStr)
	total, _ := strconv.ParseFloat(totalStr, 64)

	// Get property details
	property, err := get_listing_by_id(propertyID)
	if err != nil || property == nil {
		http.Error(w, "Property not found", http.StatusNotFound)
		return
	}

	// Prepare template data
	templateData := struct {
		Auth     AuthContext
		Property *Listing
		Nights   int
		Total    float64
	}{
		Auth:     authCtx,
		Property: property,
		Nights:   nights,
		Total:    total,
	}

	tmpl := template.Must(template.ParseFiles("template/booking_success.html"))
	err = tmpl.Execute(w, templateData)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error executing template:", err)
	}
}

func enable_review_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if user is authenticated
	authenticated, hostID := is_authenticated(r)
	if !authenticated {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data
	bookingIDStr := r.FormValue("booking_id")
	if bookingIDStr == "" {
		http.Error(w, "Booking ID is required", http.StatusBadRequest)
		return
	}

	bookingID, err := strconv.Atoi(bookingIDStr)
	if err != nil {
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	// Enable review for the booking
	err = enable_review_for_booking(bookingID, hostID)
	if err != nil {
		http.Error(w, "Error enabling review: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Review enabled successfully"))
}

func submit_review_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authenticated, userID := is_authenticated(r)
	if !authenticated {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	propertyIDStr := r.FormValue("property_id")
	bookingIDStr := r.FormValue("booking_id")
	ratingStr := r.FormValue("rating")
	comment := strings.TrimSpace(r.FormValue("comment"))

	if propertyIDStr == "" || bookingIDStr == "" || ratingStr == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	propertyID, err := strconv.Atoi(propertyIDStr)
	if err != nil {
		http.Error(w, "Invalid property ID", http.StatusBadRequest)
		return
	}

	bookingID, err := strconv.Atoi(bookingIDStr)
	if err != nil {
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 1 || rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	if len(comment) < 10 {
		http.Error(w, "Review comment must be at least 10 characters long", http.StatusBadRequest)
		return
	}

	err = create_review_with_booking(propertyID, userID, bookingID, rating, comment)
	if err != nil {
		http.Error(w, "Error submitting review: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Review submitted successfully"))
}

func add_listing_handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		authenticated, _ := is_authenticated(r)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		authCtx := get_auth(r)

		templateData := struct {
			Auth AuthContext
		}{
			Auth: authCtx,
		}

		tmpl := template.Must(template.ParseFiles("template/add_listing.html"))
		err := tmpl.Execute(w, templateData)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("Error executing template:", err)
		}

	case http.MethodPost:
		authenticated, userID := is_authenticated(r)
		if !authenticated {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Parse form data
		title := strings.TrimSpace(r.FormValue("title"))
		description := strings.TrimSpace(r.FormValue("description"))
		country := strings.TrimSpace(r.FormValue("country"))
		city := strings.TrimSpace(r.FormValue("city"))
		address := strings.TrimSpace(r.FormValue("address"))
		priceStr := r.FormValue("price")
		propertyType := r.FormValue("type")

		// Amenities
		wifi := r.FormValue("wifi") == "on"
		kitchen := r.FormValue("kitchen") == "on"
		ac := r.FormValue("air_conditioning") == "on"
		parking := r.FormValue("parking") == "on"
		pool := r.FormValue("pool") == "on"
		washer := r.FormValue("washer") == "on"
		dryer := r.FormValue("dryer") == "on"
		tv := r.FormValue("tv") == "on"
		heating := r.FormValue("heating") == "on"
		balcony := r.FormValue("balcony") == "on"
		pets := r.FormValue("pets_allowed") == "on"

		// Validate required fields
		if title == "" || description == "" || country == "" || city == "" || address == "" || priceStr == "" || propertyType == "" {
			http.Error(w, "All required fields must be filled", http.StatusBadRequest)
			return
		}

		// Validate price
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price <= 0 {
			http.Error(w, "Invalid price", http.StatusBadRequest)
			return
		}

		// Validate property type
		validTypes := map[string]bool{
			"apartment": true,
			"house":     true,
			"room":      true,
			"other":     true,
		}
		if !validTypes[propertyType] {
			http.Error(w, "Invalid property type", http.StatusBadRequest)
			return
		}

		// Create the listing
		listingID, err := create_listing(userID, title, country, city, address, description, price, propertyType)
		if err != nil {
			http.Error(w, "Error creating listing: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Add amenities
		create_amenities(listingID, wifi, ac, kitchen, parking, pets, pool, washer, dryer, tv, heating, balcony)

		// Redirect to the new listing
		http.Redirect(w, r, "/property/"+strconv.Itoa(listingID), http.StatusSeeOther)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	http.HandleFunc("/property/", property_detail_handler)
	http.HandleFunc("/add-listing", add_listing_handler)

	http.HandleFunc("/book", booking_handler)
	http.HandleFunc("/booking-success", booking_success_handler)

	http.HandleFunc("/enable-review", enable_review_handler)
	http.HandleFunc("/submit-review", submit_review_handler)

	http.HandleFunc("/edit-profile", edit_profile_handler)
	http.HandleFunc("/edit-listing/", edit_listing_handler)
	http.HandleFunc("/delete-listing/", delete_listing_handler)

	log.Println("Server starting on :8080")

	init_database()

	update_tables_for_reviews()

	if err := os.MkdirAll("static/uploads", 0755); err != nil {
		log.Printf("Warning: Could not create uploads directory: %v", err)
	}
	http.ListenAndServe(":8080", nil)
}
