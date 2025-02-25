package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func healthCheck(writer http.ResponseWriter, req *http.Request) {

	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte("OK"))

}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {

	currentHitcount := cfg.fileserverHits.Load()

	formatted := fmt.Sprintf("Hits: %d", currentHitcount)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(formatted))

}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {

	cfg.fileserverHits.Store(0)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.WriteHeader(http.StatusOK)

	w.Write([]byte("Reset successful"))

}

func main() {

	const port = "8080"

	mux := http.NewServeMux()

	Server := &http.Server{

		Addr:    ":" + port,
		Handler: mux,
	}

	apiCfg := apiConfig{}

	fileServer := http.FileServer(http.Dir("."))
	handler := http.StripPrefix("/app/", fileServer)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	mux.HandleFunc("GET /api/healthz", healthCheck)

	mux.HandleFunc("GET /api/metrics", apiCfg.metricsHandler)

	mux.HandleFunc("POST /api/reset", apiCfg.resetHandler)

	err := Server.ListenAndServe()

	if err != nil {
		log.Fatal(err)
	}

}
