package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "1234"
	dbname   = "sandbox"
)

var (
	db *sql.DB
	mu sync.Mutex
)

func initDB() {
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
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
		CREATE TABLE IF NOT EXISTS public.counter (
			id SERIAL PRIMARY KEY,
			value INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// Инициализация счётчика, если таблица пуста
	_, err = db.Exec(`INSERT INTO public.counter (value) VALUES (0) ON CONFLICT DO NOTHING`)
	if err != nil {
		log.Fatalf("Failed to initialize counter: %v", err)
	}

	log.Println("Database initialized successfully")
}

func getCountHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	var counter int
	err := db.QueryRow(`SELECT value FROM public.counter WHERE id = 1`).Scan(&counter)
	if err != nil {
		http.Error(w, "Failed to fetch counter value", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Current count: %d", counter)
}

func postCountHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Count int `json:"count"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mu.Lock()
	defer mu.Unlock()

	// Увеличиваем значение счётчика в базе данных
	_, err = db.Exec(`UPDATE public.counter SET value = value + $1 WHERE id = 1`, data.Count)
	if err != nil {
		http.Error(w, "Failed to update counter", http.StatusInternalServerError)
		return
	}

	var newCounter int
	err = db.QueryRow(`SELECT value FROM public.counter WHERE id = 1`).Scan(&newCounter)
	if err != nil {
		http.Error(w, "Failed to fetch updated counter value", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Count incremented by %d, new count: %d", data.Count, newCounter)
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCountHandler(w, r)
		case http.MethodPost:
			postCountHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("Server is running on http://localhost:8082")
	if err := http.ListenAndServe(":8082", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

//curl -X GET http://localhost:8082/count
//curl -X POST -H "Content-Type: application/json" -d '{"count": 5}' http://localhost:8082/count
