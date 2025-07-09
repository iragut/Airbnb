package main

import (
	"database/sql"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

const DATA_SOURCE_PATH = "root:example@(127.0.0.1:3306)/mysql-airbnb?parseTime=true"

var db *sql.DB

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

func create_user(email string, password string) {
	pass, _ := HashPassword(password)
	log.Println("Hashed password:", pass)

	query := "INSERT INTO Users (username, password, email, phone_number, role) VALUES ('admin', '" + pass + "', '" + email + "', '1234567890', 'admin');"
	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Error creating user: %v", err)
	}
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

func init_database() {
	db = connect_to_database()
	log.Println("Connected to the database successfully")

	query := "SELECT * FROM Users"
	_, err := db.Exec(query)
	if err != nil {
		log.Println("Tables does not exist, creating tables...")
		create_tables()
		create_user("test@test.com", "test")
	} else {
		log.Println("Tables exists, skipping creation.")
	}
}
