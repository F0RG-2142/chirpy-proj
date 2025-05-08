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
	"time"

	"github.com/F0RG-2142/chirpy-proj/internal/auth"
	"github.com/F0RG-2142/chirpy-proj/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	secret         string
}

var Cfg apiConfig

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	Cfg.platform = os.Getenv("PLATFORM")
	Cfg.secret = os.Getenv("JWT_SECRET")

	db, _ := sql.Open("postgres", dbURL)
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	Cfg.db = database.New(db)

	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", Cfg.middlewareMetricsInc(http.FileServer(http.Dir("./")))))
	mux.Handle("/assets", Cfg.middlewareMetricsInc(http.FileServer(http.Dir("./"))))
	mux.Handle("GET /api/healthz", Cfg.middlewareMetricsInc(http.HandlerFunc(readiness)))
	mux.Handle("GET /admin/metrics", Cfg.middlewareMetricsInc(http.HandlerFunc(metrics)))
	mux.Handle("POST /admin/reset", http.HandlerFunc(reset))
	mux.Handle("POST /api/users", http.HandlerFunc(newUser))
	mux.Handle("POST /api/reset", http.HandlerFunc(resetDb))
	mux.Handle("POST /api/login", http.HandlerFunc(login))
	mux.Handle("POST /api/yaps", http.HandlerFunc(yaps))
	mux.Handle("GET /api/yaps", http.HandlerFunc(getYaps))
	mux.Handle("GET /api/yaps/{yapId}", http.HandlerFunc(getYap))
	mux.Handle("POST /api/refresh", http.HandlerFunc(refresh))
	mux.Handle("POST /api/revoke", http.HandlerFunc(revoke))
	mux.Handle("PUT /api/users", http.HandlerFunc(update))

	server := &http.Server{Handler: mux, Addr: ":8080"}
	fmt.Println("Listening on http://localhost:8080/")
	server.ListenAndServe()
}

func update(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	//get auth token and validate
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	user_id, err := auth.ValidateJWT(token, Cfg.secret)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusUnauthorized)
		return
	}
	//decode request
	req := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		Email:    "",
		Password: "",
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	//hash passw and update user
	hashed_pass, err := auth.HashPassword(req.Password)
	if err != nil {

	}
	params := database.UpdateUserParams{
		Email:          req.Email,
		HashedPassword: hashed_pass,
		ID:             user_id,
	}
	err = Cfg.db.UpdateUser(r.Context(), params)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusFailedDependency)
		return
	}
	user, err := Cfg.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusFailedDependency)
		return
	}
	resp := struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, `{"error":"Failed to create response"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResp)
}

func revoke(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusUnauthorized)
		return
	}
	refreshToken, err := Cfg.db.GetRefreshToken(r.Context(), token)
	if err != nil {
		log.Printf("Error fetching refresh token: %v", err)
		http.Error(w, `{"error":"Invalid refresh token"}`, http.StatusUnauthorized)
		return
	}
	err = Cfg.db.RevokeRefreshToken(r.Context(), refreshToken.Token)
	if err != nil {
		http.Error(w, `"error":"Could not revoke Refresh Token"`, http.StatusFailedDependency)
	}
	w.WriteHeader(http.StatusNoContent)
}

func refresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusUnauthorized)
		return
	}
	refreshToken, err := Cfg.db.GetRefreshToken(r.Context(), token)
	if err != nil {
		log.Printf("Error fetching refresh token: %v", err)
		http.Error(w, `{"error":"Invalid refresh token"}`, http.StatusUnauthorized)
		return
	}
	if !refreshToken.RevokedAt.Valid {
		http.Error(w, `{"error":"Refresh token is revoked"}`, http.StatusUnauthorized)
		return
	}
	if time.Now().After(refreshToken.ExpiresAt) {
		http.Error(w, `{"error":"Refresh token is expired"}`, http.StatusUnauthorized)
		return
	}
	tokenSecret := Cfg.secret
	if tokenSecret == "" {
		log.Println("JWT_SECRET not set")
		http.Error(w, `{"error":"Server configuration error"}`, http.StatusInternalServerError)
		return
	}
	accessToken, err := auth.MakeJWT(refreshToken.UserID, tokenSecret, time.Hour)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		http.Error(w, `{"error":"Failed to generate access token"}`, http.StatusInternalServerError)
		return
	}
	resp := struct {
		AccessToken string `json:"token"`
	}{
		AccessToken: accessToken,
	}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		http.Error(w, `{"error":"Failed to create response"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResp)
}

func login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	//parse req
	req := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		Email:    "",
		Password: "",
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	//verify usern and passw
	user, err := Cfg.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, `{"error":"Incorrect username or password"}`, http.StatusBadRequest)
	}
	err = auth.CheckPasswordHash(user.HashedPassword, req.Password)
	if err != nil {

		http.Error(w, `{"error":"Incorrect username or password"}`, http.StatusBadRequest)
	}
	//make jwt
	Token, err := auth.MakeJWT(user.ID, Cfg.secret, time.Hour)
	if err != nil {
		log.Printf("Error generating JWT for user %q: %v", user.ID, err)
		http.Error(w, `{"error":"Failed to generate access token"}`, http.StatusInternalServerError)
		return
	}
	refreshToken, _ := auth.MakeRefreshToken()
	params := database.NewRefreshTokenParams{
		Token:  refreshToken,
		UserID: user.ID,
	}
	Cfg.db.NewRefreshToken(r.Context(), params)
	resp := struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        Token,
		RefreshToken: refreshToken,
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, `{"error":"Failed to create response"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResp)
}

func getYap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, err := uuid.Parse(r.URL.Query().Get("yapId"))
	if err != nil {
		w.WriteHeader(http.StatusFailedDependency)
	}
	yap, err := Cfg.db.GetYapByID(r.Context(), uuid.UUID(id))
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
	}
	yapJSON, err := json.Marshal(yap)
	if err != nil {
		w.WriteHeader(http.StatusFailedDependency)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(yapJSON)
}

func getYaps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	yaps, err := Cfg.db.GetAllYaps(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
	}
	yapsJSON, err := json.Marshal(yaps)
	if err != nil {
		w.WriteHeader(http.StatusFailedDependency)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(yapsJSON)
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
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	//validate email and password
	if req.Email == "" {
		http.Error(w, `{"error":"Email is required"}`, http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, `{"error":"Password is required"}`, http.StatusBadRequest)
		return
	}
	//hash passw
	hashedPass, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, `{"error":"Faileed to hash password"}`, http.StatusFailedDependency)
	}
	//check if db is initialized
	if Cfg.db == nil {
		log.Println("Database not initialized")
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	//Create user and resepond with created user
	params := database.CreateUserParams{
		Email:          req.Email,
		HashedPassword: hashedPass,
	}
	user, err := Cfg.db.CreateUser(r.Context(), params)
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

func yaps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	//req struct
	var req struct {
		Body   string    `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}
	//decode req
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	if err := decoder.Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		w.WriteHeader(500)
		return
	}
	//response struct
	type returnValues struct {
		Id        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserId    string    `json:"user_id"`
		Err       string    `json:"error"`
		Valid     bool      `json:"valid"`
	}
	//get bearer token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusFailedDependency)
	}
	//validate token
	user_id, err := auth.ValidateJWT(token, Cfg.secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
	}
	if user_id != req.UserId {
		w.WriteHeader(http.StatusUnauthorized)
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
	cleaned_body := req.Body
	cleaned_body = strings.Replace(cleaned_body, "kerfuffle", "****", -1)
	cleaned_body = strings.Replace(cleaned_body, "sharbert", "****", -1)
	cleaned_body = strings.Replace(cleaned_body, "fornax", "****", -1)
	//save chirp to db
	params := database.NewYapParams{
		Body:   cleaned_body,
		UserID: req.UserId,
	}
	chirp, err := Cfg.db.NewYap(r.Context(), params)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		http.Error(w, `{"error":"Failed to create chirp"}`, http.StatusInternalServerError)
		return
	}

	respBody := returnValues{
		Id:        chirp.ID.String(),
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserId:    chirp.UserID.String(),
		Valid:     true,
	}
	//marshal and send reponse on successful creation
	data, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(http.StatusCreated)
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
