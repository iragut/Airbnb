package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

const DATA_SOURCE_PATH = "root:example@(127.0.0.1:3306)/mysql-airbnb?parseTime=true"

var db *sql.DB

type UserData struct {
	ID          int
	Username    string
	Email       string
	PhoneNumber string
	Role        string
	CreatedAt   string
	FirstName   string
	LastName    string
}

type Listing struct {
	ID          int
	UserID      int
	Title       string
	Country     string
	City        string
	Address     string
	Description string
	Price       float64
	Type        string
	ImageURL    string
	HasWifi     bool
	HasKitchen  bool
	HasAC       bool
	HasParking  bool
	CreatedAt   string
}

type SearchParams struct {
	Destination  string
	CheckIn      string
	CheckOut     string
	Guests       string
	MinPrice     float64
	MaxPrice     float64
	PropertyType string
	Page         int
	Limit        int

	Wifi            bool
	Kitchen         bool
	AirConditioning bool
	Parking         bool
	Pool            bool
	TV              bool
	Washer          bool
	Dryer           bool
	Heating         bool
	Balcony         bool
	PetsAllowed     bool
}

type ListingsResult struct {
	Listings     []Listing
	TotalResults int
	TotalPages   int
	CurrentPage  int
}

type PropertyAmenities struct {
	PostID          int
	Wifi            bool
	AirConditioning bool
	Kitchen         bool
	Parking         bool
	PetsAllowed     bool
	Pool            bool
	Washer          bool
	Dryer           bool
	TV              bool
	Heating         bool
	Balcony         bool
}

type Review struct {
	ID        int
	PostID    int
	UserID    int
	Username  string
	Rating    int
	Comment   string
	CreatedAt string
}

type PropertyDetail struct {
	Property  *Listing
	Host      *UserData
	Amenities *PropertyAmenities
	Reviews   []Review
}

type Booking struct {
	ID            int
	PostID        int
	UserID        int
	HostID        int
	StartDate     string
	EndDate       string
	Guests        int
	TotalPrice    float64
	Status        string
	CreatedAt     string
	PropertyTitle string
	PropertyCity  string
	UserName      string
}

func create_listing(user_id int, title string, country string, city string, address string, description string, price float64, postType string) (int, error) {
	query := "INSERT INTO Posts (user_id, title, country, city, address, description, price, type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"

	result, err := db.Exec(query, user_id, title, country, city, address, description, price, postType)
	if err != nil {
		log.Printf("Error creating listing: %v", err)
		return 0, err
	}

	// Get the ID of the newly created listing
	listingID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting listing ID: %v", err)
		return 0, err
	}

	log.Printf("Listing created successfully with ID: %d", listingID)
	return int(listingID), nil
}

func get_countries() []string {
	return []string{
		"United States",
		"United Kingdom",
		"France",
		"Germany",
		"Italy",
		"Spain",
		"Japan",
		"Australia",
		"Canada",
		"Netherlands",
		"Switzerland",
		"Austria",
		"Belgium",
		"Portugal",
		"Greece",
		"Croatia",
		"Czech Republic",
		"Poland",
		"Hungary",
		"Romania",
		"UAE",
		"Thailand",
		"Indonesia",
		"Malaysia",
		"Singapore",
		"South Korea",
		"China",
		"India",
		"Brazil",
		"Mexico",
		"Argentina",
		"Chile",
		"Colombia",
	}
}

func create_booking(postID, userID, hostID, guests int, startDate, endDate string, totalPrice float64) error {
	// First check if dates are available
	available, err := check_availability(postID, startDate, endDate)
	if err != nil {
		return err
	}
	if !available {
		return fmt.Errorf("dates are not available")
	}

	query := `INSERT INTO Bookings (post_id, user_id, host_id, start_date, end_date, guests, total_price) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = db.Exec(query, postID, userID, hostID, startDate, endDate, guests, totalPrice)
	if err != nil {
		log.Printf("Error creating booking: %v", err)
		return err
	}

	log.Printf("Booking created successfully for user %d, property %d", userID, postID)
	return nil
}

func check_availability(postID int, startDate, endDate string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM Bookings 
		WHERE post_id = ? 
		AND (
			(start_date <= ? AND end_date > ?) OR
			(start_date < ? AND end_date >= ?) OR
			(start_date >= ? AND end_date <= ?)
		)`

	var count int
	err := db.QueryRow(query, postID, startDate, startDate, endDate, endDate, startDate, endDate).Scan(&count)
	if err != nil {
		log.Printf("Error checking availability: %v", err)
		return false, err
	}

	return count == 0, nil
}

func get_user_bookings(userID int) ([]Booking, error) {
	var bookings []Booking

	query := `
		SELECT b.id, b.post_id, b.user_id, b.host_id, b.start_date, b.end_date, 
		       b.guests, b.total_price, b.created_at, p.title, p.city
		FROM Bookings b
		JOIN Posts p ON b.post_id = p.id
		WHERE b.user_id = ?
		ORDER BY b.created_at DESC`

	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error fetching user bookings: %v", err)
		return bookings, err
	}
	defer rows.Close()

	for rows.Next() {
		var booking Booking
		err := rows.Scan(
			&booking.ID, &booking.PostID, &booking.UserID, &booking.HostID,
			&booking.StartDate, &booking.EndDate, &booking.Guests, &booking.TotalPrice,
			&booking.CreatedAt, &booking.PropertyTitle, &booking.PropertyCity,
		)
		if err != nil {
			log.Printf("Error scanning booking: %v", err)
			continue
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

func get_host_bookings(hostID int) ([]Booking, error) {
	var bookings []Booking

	query := `
		SELECT b.id, b.post_id, b.user_id, b.host_id, b.start_date, b.end_date, 
		       b.guests, b.total_price, b.created_at, p.title, p.city, u.username
		FROM Bookings b
		JOIN Posts p ON b.post_id = p.id
		JOIN Users u ON b.user_id = u.id
		WHERE b.host_id = ?
		ORDER BY b.created_at DESC`

	rows, err := db.Query(query, hostID)
	if err != nil {
		log.Printf("Error fetching host bookings: %v", err)
		return bookings, err
	}
	defer rows.Close()

	for rows.Next() {
		var booking Booking
		err := rows.Scan(
			&booking.ID, &booking.PostID, &booking.UserID, &booking.HostID,
			&booking.StartDate, &booking.EndDate, &booking.Guests, &booking.TotalPrice,
			&booking.CreatedAt, &booking.PropertyTitle, &booking.PropertyCity, &booking.UserName,
		)
		if err != nil {
			log.Printf("Error scanning host booking: %v", err)
			continue
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

func calculate_booking_price(propertyID int, startDate, endDate string) (float64, int, error) {
	// Get property price
	property, err := get_listing_by_id(propertyID)
	if err != nil || property == nil {
		return 0, 0, fmt.Errorf("property not found")
	}

	// Parse dates and calculate nights
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0, 0, err
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return 0, 0, err
	}

	nights := int(end.Sub(start).Hours() / 24)
	if nights <= 0 {
		return 0, 0, fmt.Errorf("invalid date range")
	}

	totalPrice := float64(nights) * property.Price

	return totalPrice, nights, nil
}

func create_amenities(post_id int, wifi, ac, kitchen, parking, pets, pool, washer, dryer, tv, heating, balcony bool) {
	query := `INSERT INTO Amenities (post_id, wifi, air_conditioning, kitchen, parking, pets_allowed, pool, washer, dryer, tv, heating, balcony) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query, post_id, wifi, ac, kitchen, parking, pets, pool, washer, dryer, tv, heating, balcony)
	if err != nil {
		log.Printf("Error creating amenities: %v", err)
	}
}
func create_review(post_id, user_id, rating int, comment string) {
	query := `INSERT INTO Reviews (post_id, user_id, rating, comment) VALUES (?, ?, ?, ?)`

	_, err := db.Exec(query, post_id, user_id, rating, comment)
	if err != nil {
		log.Printf("Error creating review: %v", err)
	}
}

func get_property_detail(propertyID int) (*PropertyDetail, error) {
	// Get basic property info
	property, err := get_listing_by_id(propertyID)
	if err != nil || property == nil {
		return nil, err
	}

	// Get host information
	host := get_user_data(property.UserID)
	if host == nil {
		host = &UserData{Username: "Unknown Host"}
	}

	// Get amenities
	amenitiesQuery := `
		SELECT wifi, air_conditioning, kitchen, parking, pets_allowed, pool, washer, dryer, tv, heating, balcony
		FROM Amenities WHERE post_id = ?`

	var amenities PropertyAmenities
	err = db.QueryRow(amenitiesQuery, propertyID).Scan(
		&amenities.Wifi, &amenities.AirConditioning, &amenities.Kitchen, &amenities.Parking,
		&amenities.PetsAllowed, &amenities.Pool, &amenities.Washer, &amenities.Dryer,
		&amenities.TV, &amenities.Heating, &amenities.Balcony,
	)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error fetching amenities: %v", err)
	}

	// Get reviews
	reviewsQuery := `
		SELECT r.id, r.post_id, r.user_id, u.username, r.rating, r.comment, r.created_at
		FROM Reviews r
		JOIN Users u ON r.user_id = u.id
		WHERE r.post_id = ?
		ORDER BY r.created_at DESC`

	rows, err := db.Query(reviewsQuery, propertyID)
	if err != nil {
		log.Printf("Error fetching reviews: %v", err)
		return nil, err
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var review Review
		err := rows.Scan(&review.ID, &review.PostID, &review.UserID, &review.Username,
			&review.Rating, &review.Comment, &review.CreatedAt)
		if err != nil {
			log.Printf("Error scanning review: %v", err)
			continue
		}
		reviews = append(reviews, review)
	}

	return &PropertyDetail{
		Property:  property,
		Host:      host,
		Amenities: &amenities,
		Reviews:   reviews,
	}, nil
}

func search_listings(params SearchParams) (*ListingsResult, error) {
	var listings []Listing
	var totalCount int

	whereConditions := []string{"1=1"}
	args := []interface{}{}
	countArgs := []interface{}{}

	// Basic search filters
	if params.Destination != "" {
		whereConditions = append(whereConditions, "(p.city LIKE ? OR p.country LIKE ? OR p.title LIKE ?)")
		searchTerm := "%" + params.Destination + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
		countArgs = append(countArgs, searchTerm, searchTerm, searchTerm)
	}

	if params.MinPrice > 0 {
		whereConditions = append(whereConditions, "p.price >= ?")
		args = append(args, params.MinPrice)
		countArgs = append(countArgs, params.MinPrice)
	}

	if params.MaxPrice > 0 {
		whereConditions = append(whereConditions, "p.price <= ?")
		args = append(args, params.MaxPrice)
		countArgs = append(countArgs, params.MaxPrice)
	}

	if params.PropertyType != "" {
		whereConditions = append(whereConditions, "p.type = ?")
		args = append(args, params.PropertyType)
		countArgs = append(countArgs, params.PropertyType)
	}

	// Amenity filters - only add conditions if amenities are requested
	amenityConditions := []string{}

	if params.Wifi {
		amenityConditions = append(amenityConditions, "a.wifi = true")
	}
	if params.Kitchen {
		amenityConditions = append(amenityConditions, "a.kitchen = true")
	}
	if params.AirConditioning {
		amenityConditions = append(amenityConditions, "a.air_conditioning = true")
	}
	if params.Parking {
		amenityConditions = append(amenityConditions, "a.parking = true")
	}
	if params.Pool {
		amenityConditions = append(amenityConditions, "a.pool = true")
	}
	if params.TV {
		amenityConditions = append(amenityConditions, "a.tv = true")
	}
	if params.Washer {
		amenityConditions = append(amenityConditions, "a.washer = true")
	}
	if params.Dryer {
		amenityConditions = append(amenityConditions, "a.dryer = true")
	}
	if params.Heating {
		amenityConditions = append(amenityConditions, "a.heating = true")
	}
	if params.Balcony {
		amenityConditions = append(amenityConditions, "a.balcony = true")
	}
	if params.PetsAllowed {
		amenityConditions = append(amenityConditions, "a.pets_allowed = true")
	}

	// Build the complete WHERE clause
	whereClause := strings.Join(whereConditions, " AND ")

	// Determine if we need to JOIN with amenities table
	joinClause := ""
	if len(amenityConditions) > 0 {
		joinClause = "INNER JOIN Amenities a ON p.id = a.post_id"
		whereClause += " AND " + strings.Join(amenityConditions, " AND ")
	} else {
		joinClause = "LEFT JOIN Amenities a ON p.id = a.post_id"
	}

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT p.id) 
		FROM Posts p
		%s
		WHERE %s`, joinClause, whereClause)

	err := db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		log.Printf("Error counting listings: %v", err)
		return nil, err
	}

	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	offset := (params.Page - 1) * params.Limit
	totalPages := (totalCount + params.Limit - 1) / params.Limit

	// Main query
	query := fmt.Sprintf(`
		SELECT p.id, p.user_id, p.title, p.country, p.city, p.address, 
		       p.description, p.price, p.type, p.created_at,
		       COALESCE(MIN(i.image_url), '') as image_url,
		       COALESCE(MAX(a.wifi), false) as has_wifi,
		       COALESCE(MAX(a.kitchen), false) as has_kitchen,
		       COALESCE(MAX(a.air_conditioning), false) as has_ac,
		       COALESCE(MAX(a.parking), false) as has_parking
		FROM Posts p
		LEFT JOIN Images i ON p.id = i.post_id
		%s
		WHERE %s
		GROUP BY p.id, p.user_id, p.title, p.country, p.city, p.address, p.description, p.price, p.type, p.created_at
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?`, joinClause, whereClause)

	args = append(args, params.Limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying listings: %v", err)
		return nil, err
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

	return &ListingsResult{
		Listings:     listings,
		TotalResults: totalCount,
		TotalPages:   totalPages,
		CurrentPage:  params.Page,
	}, nil
}

func get_listing_by_id(listingID int) (*Listing, error) {
	query := `
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
		WHERE p.id = ?
		GROUP BY p.id, p.user_id, p.title, p.country, p.city, p.address, p.description, p.price, p.type, p.created_at
		LIMIT 1`

	var listing Listing
	err := db.QueryRow(query, listingID).Scan(
		&listing.ID, &listing.UserID, &listing.Title, &listing.Country,
		&listing.City, &listing.Address, &listing.Description,
		&listing.Price, &listing.Type, &listing.CreatedAt,
		&listing.ImageURL, &listing.HasWifi, &listing.HasKitchen,
		&listing.HasAC, &listing.HasParking,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Printf("Error querying listing by ID: %v", err)
		return nil, err
	}

	return &listing, nil
}

func get_post_count_by_city(city string) int {
	query := `SELECT COUNT(*) FROM Posts WHERE city = ?`
	var count int
	err := db.QueryRow(query, city).Scan(&count)

	if err != nil {
		log.Printf("Error querying post count for city %s: %v", city, err)
		return 0
	}

	return count
}

func get_user_data(user_id int) *UserData {
	query := `
		SELECT u.id, u.username, u.email, u.phone_number, u.role, u.created_at,
		       COALESCE(p.first_name, '') as first_name, 
		       COALESCE(p.last_name, '') as last_name
		FROM Users u
		LEFT JOIN PersonalData p ON u.id = p.user_id
		WHERE u.id = ?
	`

	var user UserData
	err := db.QueryRow(query, user_id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PhoneNumber,
		&user.Role, &user.CreatedAt, &user.FirstName, &user.LastName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User not found with ID: %d", user_id)
			return nil
		}
		log.Printf("Error querying user data: %v", err)
		return nil
	}

	return &user
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func create_tables() {
	content, err := os.ReadFile("sql_tables/tables")
	if err != nil {
		log.Fatal("Error reading file:", err)
	}

	tables := strings.Split(string(content), ";")
	for i := 0; i < len(tables)-1; i++ {

		_, err := db.Exec(tables[i])
		if err != nil {
			log.Fatalf("Error creating table: %s, %v", tables[i], err)
		}
		log.Printf("Table created successfully: %s", tables[i])
	}
}

func create_user(email string, password string, username string, phone_number string, role string) {
	pass, _ := HashPassword(password)
	log.Println("Hashed password:", pass)

	query := "INSERT INTO Users (username, password, email, phone_number, role) VALUES (?, ?, ?, ?, ?)"
	_, err := db.Exec(query, username, pass, email, phone_number, role)
	if err != nil {
		log.Fatalf("Error creating user: %v", err)
	}
}

func create_post(user_id int, title string, country string, city string, address string, description string, price float64, postType string) {
	query := "INSERT INTO Posts (user_id, title, country, city, address, description, price, type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	_, err := db.Exec(query, user_id, title, country, city, address, description, price, postType)
	if err != nil {
		log.Fatalf("Error creating post: %v", err)
	}
	log.Println("Post created successfully")
}

func enable_review_for_booking(bookingID int, hostID int) error {
	query := `SELECT COUNT(*) FROM Bookings WHERE id = ? AND host_id = ?`
	var count int
	err := db.QueryRow(query, bookingID, hostID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("booking not found or you're not the host")
	}

	updateQuery := `UPDATE Bookings SET review_enabled = true WHERE id = ? AND host_id = ?`
	_, err = db.Exec(updateQuery, bookingID, hostID)
	if err != nil {
		log.Printf("Error enabling review: %v", err)
		return err
	}

	log.Printf("Review enabled for booking %d by host %d", bookingID, hostID)
	return nil
}

func can_user_review_property(userID, propertyID int) (bool, int, error) {
	query := `
		SELECT id FROM Bookings 
		WHERE user_id = ? AND post_id = ? AND review_enabled = true
		AND id NOT IN (SELECT COALESCE(booking_id, 0) FROM Reviews WHERE user_id = ? AND post_id = ?)
		LIMIT 1`

	var bookingID int
	err := db.QueryRow(query, userID, propertyID, userID, propertyID).Scan(&bookingID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, 0, nil
		}
		return false, 0, err
	}

	return true, bookingID, nil
}

func create_review_with_booking(postID, userID, bookingID, rating int, comment string) error {
	canReview, _, err := can_user_review_property(userID, postID)
	if err != nil {
		return err
	}
	if !canReview {
		return fmt.Errorf("you don't have permission to review this property")
	}

	query := `INSERT INTO Reviews (post_id, user_id, booking_id, rating, comment) VALUES (?, ?, ?, ?, ?)`
	_, err = db.Exec(query, postID, userID, bookingID, rating, comment)
	if err != nil {
		log.Printf("Error creating review: %v", err)
		return err
	}

	log.Printf("Review created for property %d by user %d", postID, userID)
	return nil
}

func get_host_bookings_for_review_management(hostID int) ([]Booking, error) {
	var bookings []Booking

	query := `
		SELECT b.id, b.post_id, b.user_id, b.host_id, b.start_date, b.end_date, 
		       b.guests, b.total_price, b.created_at, p.title, p.city, u.username,
		       COALESCE(b.review_enabled, false) as review_enabled,
		       CASE WHEN r.id IS NOT NULL THEN true ELSE false END as has_review
		FROM Bookings b
		JOIN Posts p ON b.post_id = p.id
		JOIN Users u ON b.user_id = u.id
		LEFT JOIN Reviews r ON b.id = r.booking_id
		WHERE b.host_id = ?
		ORDER BY b.created_at DESC`

	rows, err := db.Query(query, hostID)
	if err != nil {
		log.Printf("Error fetching host bookings for review management: %v", err)
		return bookings, err
	}
	defer rows.Close()

	for rows.Next() {
		var booking Booking
		var reviewEnabled, hasReview bool
		err := rows.Scan(
			&booking.ID, &booking.PostID, &booking.UserID, &booking.HostID,
			&booking.StartDate, &booking.EndDate, &booking.Guests, &booking.TotalPrice,
			&booking.CreatedAt, &booking.PropertyTitle, &booking.PropertyCity, &booking.UserName,
			&reviewEnabled, &hasReview,
		)
		if err != nil {
			log.Printf("Error scanning host booking: %v", err)
			continue
		}

		// Add custom fields for review management
		if reviewEnabled {
			booking.Status = "review_enabled"
		} else {
			booking.Status = "completed"
		}

		bookings = append(bookings, booking)
	}

	return bookings, nil
}

func get_user_reviewable_bookings(userID int) ([]Booking, error) {
	var bookings []Booking

	query := `
		SELECT b.id, b.post_id, b.user_id, b.host_id, b.start_date, b.end_date, 
		       b.guests, b.total_price, b.created_at, p.title, p.city,
		       CASE WHEN r.id IS NOT NULL THEN true ELSE false END as has_review
		FROM Bookings b
		JOIN Posts p ON b.post_id = p.id
		LEFT JOIN Reviews r ON b.id = r.booking_id
		WHERE b.user_id = ? AND COALESCE(b.review_enabled, false) = true
		ORDER BY b.created_at DESC`

	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error fetching user reviewable bookings: %v", err)
		return bookings, err
	}
	defer rows.Close()

	for rows.Next() {
		var booking Booking
		var hasReview bool
		err := rows.Scan(
			&booking.ID, &booking.PostID, &booking.UserID, &booking.HostID,
			&booking.StartDate, &booking.EndDate, &booking.Guests, &booking.TotalPrice,
			&booking.CreatedAt, &booking.PropertyTitle, &booking.PropertyCity, &hasReview,
		)
		if err != nil {
			log.Printf("Error scanning reviewable booking: %v", err)
			continue
		}

		if hasReview {
			booking.Status = "reviewed"
		} else {
			booking.Status = "can_review"
		}

		bookings = append(bookings, booking)
	}

	return bookings, nil
}

func update_tables_for_reviews() {
	alterBookings := `ALTER TABLE Bookings ADD COLUMN review_enabled BOOLEAN DEFAULT FALSE`
	_, err := db.Exec(alterBookings)
	if err != nil {
		log.Printf("Note: review_enabled column might already exist: %v", err)
	}

	alterReviews := `ALTER TABLE Reviews ADD COLUMN booking_id INT NULL`
	_, err = db.Exec(alterReviews)
	if err != nil {
		log.Printf("Note: booking_id column might already exist: %v", err)
	}

	addFK := `ALTER TABLE Reviews ADD CONSTRAINT fk_reviews_booking 
			  FOREIGN KEY (booking_id) REFERENCES Bookings(id) ON DELETE SET NULL`
	_, err = db.Exec(addFK)
	if err != nil {
		log.Printf("Note: foreign key constraint might already exist: %v", err)
	}

	log.Println("Tables updated for review system")
}

func create_posts_test_data() {
	create_post(1, "Cozy Apartment", "USA", "New York", "123 Broadway St", "A cozy apartment in the city center with modern amenities and great city views.", 100.0, "apartment")
	create_amenities(1, true, true, true, false, false, false, true, true, true, true, false)
	create_review(1, 1, 5, "Amazing apartment! Perfect location and very clean. The host was super responsive and helpful.")
	create_review(1, 1, 4, "Great place to stay in NYC. Kitchen was well-equipped and the bed was comfortable. Would recommend!")

	create_post(1, "Luxury Loft", "Indonesia", "Bali", "456 Beach Road", "A luxury loft with stunning ocean views, private balcony, and modern tropical design.", 150.0, "apartment")
	create_amenities(2, true, true, true, true, false, true, true, false, true, false, true)
	create_review(2, 1, 5, "Absolutely stunning! The ocean view is breathtaking and the loft is beautifully designed. Perfect for a romantic getaway.")
	create_review(2, 1, 5, "Best vacation rental we've ever stayed at. The pool area is amazing and the location is unbeatable.")

	create_post(1, "Beach House", "Spain", "Barcelona", "789 Coastal Ave", "A beautiful beach house with direct ocean access, perfect for families and groups.", 250.0, "house")
	create_amenities(3, true, false, true, true, true, false, true, true, true, false, true)
	create_review(3, 1, 4, "Perfect family vacation spot! Kids loved being so close to the beach. House has everything you need.")

	create_post(1, "Mountain Retreat", "Japan", "Tokyo", "321 Mountain Path", "A unique mountain retreat just outside Tokyo, offering peace and tranquility with city access.", 200.0, "house")
	create_amenities(4, true, false, true, true, false, false, false, false, true, true, false)
	create_review(4, 1, 5, "What a unique find! Perfect escape from the city while still being accessible. Very peaceful and well-maintained.")
	create_review(4, 1, 4, "Great for a digital detox. Beautiful surroundings and the host provided excellent local recommendations.")

	create_post(1, "Luxury Villa", "France", "Paris", "654 Champs Elysees", "An elegant Parisian villa with private pool, garden, and classic French architecture.", 500.0, "house")
	create_amenities(5, true, true, true, true, false, true, true, true, true, true, true)
	create_review(5, 1, 5, "Pure luxury! Felt like staying in a high-end hotel. The pool and garden are magnificent. Worth every euro!")

	create_post(1, "Modern City Loft", "UK", "London", "987 Thames St", "A sleek modern loft in the heart of London with industrial design and city views.", 300.0, "apartment")
	create_amenities(6, true, true, true, false, false, false, true, true, true, true, false)
	create_review(6, 1, 4, "Fantastic location and beautiful modern design. Walking distance to all major attractions. Highly recommend!")
	create_review(6, 1, 5, "Stylish and comfortable. The loft has a great vibe and the host was incredibly welcoming.")

	create_post(1, "Tuscan-Style Cottage", "Italy", "Rome", "147 Villa Road", "A charming cottage with authentic Italian character, beautiful gardens, and peaceful countryside views.", 180.0, "house")
	create_amenities(7, true, false, true, true, true, false, false, false, true, true, true)
	create_review(7, 1, 5, "Like staying in a fairytale! The cottage is beautifully decorated and the garden is perfect for morning coffee.")

	create_post(1, "Sky-High Penthouse", "UAE", "Dubai", "258 Burj St", "A luxurious penthouse suite with panoramic city views, premium amenities, and world-class service.", 400.0, "apartment")
	create_amenities(8, true, true, true, true, false, true, true, true, true, true, true)
	create_review(8, 1, 5, "Absolutely incredible! The views are out of this world. Felt like a VIP the entire stay. Perfect for special occasions.")
	create_review(8, 1, 4, "Stunning apartment with amazing amenities. The infinity pool on the rooftop is unforgettable.")

	create_post(1, "Historic Mansion", "France", "Paris", "369 Historic Blvd", "A beautifully restored 18th-century mansion with original details, elegant furnishings, and rich history.", 600.0, "house")
	create_amenities(9, true, true, true, true, false, false, true, true, true, true, true)
	create_review(9, 1, 5, "Staying here is like living in a museum! Incredible history and the restoration is flawless. A truly unique experience.")

	log.Println("Created test posts and amenities successfully")
}

func connect_to_database() *sql.DB {
	db, err := sql.Open("mysql", DATA_SOURCE_PATH)
	if err != nil {
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	return db
}

func get_user_id(email string) int {
	query := "SELECT id FROM Users WHERE email = ?"
	var userID int
	err := db.QueryRow(query, email).Scan(&userID)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("User not found:", email)
			return 0
		}
		log.Println("Error querying user ID:", err)
		return 0
	}

	return userID
}

func check_user_exists(email string, password string) bool {
	query := "SELECT password FROM Users WHERE email = ?"

	var storedPassword string
	err := db.QueryRow(query, email).Scan(&storedPassword)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("User not found:", email)
			return false
		}
		log.Println("Error querying user:", err)
		return false
	}

	return CheckPasswordHash(password, storedPassword)

}

func get_current_user_id(r *http.Request) int {
	session, _ := store.Get(r, "cookie-name")

	auth, ok := session.Values["authenticated"].(bool)
	if !ok || !auth {
		return 0
	}

	user_id, ok := session.Values["user_id"].(int)
	if !ok {
		return 0
	}

	return user_id
}

func init_database() {
	db = connect_to_database()
	log.Println("Connected to the database successfully")

	query := "SELECT * FROM Users"
	_, err := db.Exec(query)
	if err != nil {
		log.Println("Tables does not exist, creating tables...")
		create_tables()
		create_user("test@test.com", "test", "Test User", "1234567890", "admin")
		create_posts_test_data()
	} else {
		log.Println("Tables exists, skipping creation.")
	}
}
