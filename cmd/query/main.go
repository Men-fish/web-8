package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "1234"
	dbname   = "sandbox"
)

var db *sql.DB

func initDB() {
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatalf("Database is not reachable: %v", err)
	}

	// Создаём таблицу, если её нет
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS public.users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	log.Println("Database initialized successfully")
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Parameter 'name' is required", http.StatusBadRequest)
		return
	}

	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM public.users WHERE name = $1)`, name).Scan(&exists)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, fmt.Sprintf("User '%s' not found", name), http.StatusNotFound)
		return
	}

	response := fmt.Sprintf("Hello, %s!", name)
	w.Write([]byte(response))
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/api/user", userHandler)

	fmt.Println("Server is running on http://localhost:8083")
	if err := http.ListenAndServe(":8083", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
// curl "http://localhost:8083/api/user?name=Men-fish"
// INSERT INTO public.users (name) VALUES ('Men-fish');
