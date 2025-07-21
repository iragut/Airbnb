package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

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

// Extract number of posts in a city
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

// Updated test data creation with all required fields
func create_posts_test_data() {
	create_post(1, "Cozy Apartment", "USA", "New York", "123 Broadway St", "A cozy apartment in the city center.", 100.0, "apartment")
	create_post(1, "Luxury Loft", "Indonesia", "Bali", "456 Beach Road", "A luxury loft with ocean views.", 150.0, "apartment")
	create_post(1, "Beach House", "Spain", "Barcelona", "789 Coastal Ave", "A beautiful beach house with ocean view.", 250.0, "house")
	create_post(1, "Mountain Cabin", "Japan", "Tokyo", "321 Mountain Path", "A rustic cabin in the mountains.", 200.0, "house")
	create_post(1, "Luxury Villa", "France", "Paris", "654 Champs Elysees", "A luxury villa with private pool.", 500.0, "house")
	create_post(1, "City Loft", "UK", "London", "987 Thames St", "A modern loft in the heart of the city.", 300.0, "apartment")
	create_post(1, "Countryside Cottage", "Italy", "Rome", "147 Villa Road", "A charming cottage in the countryside.", 180.0, "house")
	create_post(1, "Penthouse Suite", "UAE", "Dubai", "258 Burj St", "A penthouse suite with stunning views.", 400.0, "apartment")
	create_post(1, "Historic Mansion", "France", "Paris", "369 Historic Blvd", "A historic mansion with rich history.", 600.0, "house")
	log.Println("Test posts created successfully")
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
