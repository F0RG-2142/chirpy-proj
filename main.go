package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/F0RG-2142/chirpy-proj/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

var Cfg apiConfig

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, _ := sql.Open("postgres", dbURL)
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	Cfg.db = database.New(db)
	Cfg.platform = platform

	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", Cfg.middlewareMetricsInc(http.FileServer(http.Dir("./")))))
	mux.Handle("/assets", Cfg.middlewareMetricsInc(http.FileServer(http.Dir("./"))))
	mux.Handle("GET /api/healthz", Cfg.middlewareMetricsInc(http.HandlerFunc(readiness)))
	mux.Handle("GET /admin/metrics", Cfg.middlewareMetricsInc(http.HandlerFunc(metrics)))
	mux.Handle("POST /admin/reset", http.HandlerFunc(reset))
	mux.Handle("POST /api/validate_chirp", http.HandlerFunc(validate_chirp))
	mux.Handle("POST /api/users", http.HandlerFunc(newUser))
	mux.Handle("POST /api/reset", http.HandlerFunc(resetDb))

	server := &http.Server{Handler: mux, Addr: ":8080"}
	server.ListenAndServe()
}

func resetDb(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if Cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
	}
	Cfg.db.DeleteAllUsers(r.Context())
}

func newUser(w http.ResponseWriter, r *http.Request) {
	//decode request body
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	//validate email
	if req.Email == "" {
		http.Error(w, `{"error":"Email is required"}`, http.StatusBadRequest)
		return
	}
	//check if db is initialized
	if Cfg.db == nil {
		log.Println("Database not initialized")
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	//Create user and resepond with created user
	user, err := Cfg.db.CreateUser(r.Context(), sql.NullString{String: req.Email, Valid: true})
	if err != nil {
		log.Printf("Error creating user: %v", err)
		http.Error(w, `{"error":"Failed to create user"}`, http.StatusInternalServerError)
		return
	}
	userJSON, err := json.Marshal(user)
	if err != nil {
		log.Printf("Error marshalling user to JSON: %v", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(userJSON)
}

func reset(w http.ResponseWriter, r *http.Request) {
	Cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Fileserver hits reset to 0"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func validate_chirp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err := decoder.Decode(&req)
	fmt.Println(req)
	if err != nil {
		log.Printf("Error decoding request: %v", err)
		w.WriteHeader(500)
		return
	}
	type returnValues struct {
		Body  string `json:"body"`
		Err   string `json:"error"`
		Valid bool   `json:"valid"`
	}
	//If body too long (>140) return error
	if len(req.Body) > 140 {
		respBody := returnValues{
			Err: "Chirp is too long",
		}
		data, err := json.Marshal(respBody)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(400)
		w.Write(data)
		return
	}
	//Clean profanities 1.0
	var cleaned_body string
	if strings.Contains(req.Body, "kerfuffle") || strings.Contains(req.Body, "sharbert") || strings.Contains(req.Body, "fornax") {
		cleaned_body = req.Body
		cleaned_body = strings.Replace(cleaned_body, "kerfuffle", "****", -1)
		cleaned_body = strings.Replace(cleaned_body, "sharbert", "****", -1)
		cleaned_body = strings.Replace(cleaned_body, "fornax", "****", -1)
	}
	respBody := returnValues{
		Body:  cleaned_body,
		Valid: true,
	}
	data, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Write(data)
}

func metrics(w http.ResponseWriter, r *http.Request) {
	hits := int(Cfg.fileserverHits.Load())
	w.WriteHeader(200)
	tmpl, err := template.ParseFiles("./metrics.html")
	if err != nil {
		fmt.Println("Error loading template:", err)
		return
	}
	data := struct {
		Count int
	}{
		Count: hits,
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(""))
}
