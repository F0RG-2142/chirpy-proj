package main

import (
	"fmt"
	"html/template"
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
