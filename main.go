package main

import (
	"encoding/json"
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

//decode:

func validationHandler(writer http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		writer.WriteHeader(500)
		return
	}

	type returnVals struct {
		Valid bool   `json:"valid,omitempty"`
		Error string `json:"error,omitempty"`
	}

	respBody := returnVals{}
	writer.Header().Set("Content-Type", "application/json")

	if len(params.Body) > 140 {
		respBody.Error = "Chirp is too long"
		dat, err := json.Marshal(respBody)
		if err != nil {
			writer.WriteHeader(500)
			writer.Write([]byte(`{"error": "Something went wrong"}`))
			return
		}
		writer.WriteHeader(400)
		writer.Write(dat)
		return
	}

	respBody.Valid = true
	dat, err := json.Marshal(respBody)
	if err != nil {
		writer.WriteHeader(500)
		writer.Write([]byte(`{"error": "Something went wrong"}`))
		return
	}

	writer.WriteHeader(200)
	writer.Write(dat)
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {

	currentHitcount := cfg.fileserverHits.Load()

	formatted := fmt.Sprintf("<html>\n<body>\n<h1>\nWelcome, Chirpy Admin</h1>\n<p>Chirpy has been visited %d times!</p>\n</body>\n</html>", currentHitcount)

	//fmt.Sprintf("Hits: %d", currentHitcount)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

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

	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)

	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)

	mux.HandleFunc("POST /api/validate_chirp", validationHandler)

	err := Server.ListenAndServe()

	if err != nil {
		log.Fatal(err)
	}

}
