package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/OleDH/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits  atomic.Int32
	databaseQueries *database.Queries
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

func sanitize(input string) string {

	substrings := strings.Split(input, " ")

	for i, v := range substrings {

		compareString := strings.ToLower(v)

		if compareString == "kerfuffle" || compareString == "sharbert" || compareString == "fornax" {

			substrings[i] = "****"

		}

	}
	output := strings.Join(substrings, " ")

	return output

}

//decode:

//todo: implement these helper functions:
//respondWithError(w http.ResponseWriter, code int, msg string)
//respondWithJSON(w http.ResponseWriter, code int, payload interface{})
//Please refactor, for me

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
		Cleaned_body string `json:"cleaned_body,omitempty"`
		Error        string `json:"error,omitempty"`
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

	respBody.Cleaned_body = sanitize(params.Body)

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

	godotenv.Load()

	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)

	dbQueries := database.New(db)

	const port = "8080"

	mux := http.NewServeMux()

	Server := &http.Server{

		Addr:    ":" + port,
		Handler: mux,
	}

	apiCfg := apiConfig{

		databaseQueries: dbQueries,
	}

	fileServer := http.FileServer(http.Dir("."))
	handler := http.StripPrefix("/app/", fileServer)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	mux.HandleFunc("GET /api/healthz", healthCheck)

	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)

	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)

	mux.HandleFunc("POST /api/validate_chirp", validationHandler)

	err = Server.ListenAndServe()

	if err != nil {
		log.Fatal(err)
	}

}
