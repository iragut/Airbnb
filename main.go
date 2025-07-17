package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tmpl := template.Must(template.ParseFiles("template/explore_page.html"))
	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error executing template:", err)
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	destination := r.FormValue("destination")
	checkin := r.FormValue("checkin")
	checkout := r.FormValue("checkout")
	guests := r.FormValue("guests")

	log.Printf("Search request: destination=%s, checkin=%s, checkout=%s, guests=%s",
		destination, checkin, checkout, guests)

	// For now, just redirect back to main page
	// Later you can create a search results page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {
	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", main_page_handler)
	http.HandleFunc("/login", login_handler)
	http.HandleFunc("/logout", logout_handler)

	http.HandleFunc("/register", register_handler)
	http.HandleFunc("/explore", explore_handler)

	http.HandleFunc("/users/", user_profile_handler)
	http.HandleFunc("/my-profile", my_profile_handler)

	log.Println("Server starting on :8080")

	init_database()
	http.ListenAndServe(":8080", nil)
}
