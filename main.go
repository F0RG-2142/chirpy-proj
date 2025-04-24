package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

var apiCfg apiConfig

func main() {
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir("./")))))
	mux.Handle("/assets", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir("./"))))
	mux.Handle("GET /api/healthz", apiCfg.middlewareMetricsInc(http.HandlerFunc(readiness)))
	mux.Handle("GET /admin/metrics", apiCfg.middlewareMetricsInc(http.HandlerFunc(metrics)))
	mux.Handle("POST /admin/reset", http.HandlerFunc(reset))
	mux.Handle("POST /api/validate_chirp", http.HandlerFunc(validate_chirp))

	server := &http.Server{Handler: mux, Addr: ":8080"}
	server.ListenAndServe()
}

func reset(w http.ResponseWriter, r *http.Request) {
	apiCfg.fileserverHits.Store(0)
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
	type chirp struct {
		Body string `json:"body"`
	}
	type return_val struct {
		Body  string `json:"body"`
		Err   string `json:"error"`
		Valid string `json:"valid"`
	}

	decoder := json.NewDecoder(r.Body)
	chirp_body := chirp{}
	err := decoder.Decode(&chirp_body)
	if err != nil {
		log.Printf("Error decoding parameters: %v", err)
		w.WriteHeader(500)
		return
	}
	if len(chirp_body.Body) > 140 {
		respBody := return_val{
			Body: chirp_body.Body,
		}
		data, err := json.Marshal(respBody)
		if err != nil {
			log.Printf("Error marshaling data: %v", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		w.Write(data)
		return
	}
	respBody := return_val{Err: "Chirp is too long"}
	data, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(500)
	w.Write(data)

}

func metrics(w http.ResponseWriter, r *http.Request) {
	hits := int(apiCfg.fileserverHits.Load())
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
