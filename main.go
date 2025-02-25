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
		// these tags indicate how the keys in the JSON should be mapped to the struct fields
		// the struct fields must be exported (start with a capital letter) if you want them parsed
		Body  string `json:"body"`
		Error int    `json:"error"`
		Valid int    `json:"valid"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		// an error will be thrown if the JSON is invalid or has the wrong types
		// any missing fields will simply have their values in the struct set to their zero value
		log.Printf("Error decoding parameters: %s", err)
		writer.WriteHeader(500)
		return
	}
	// params is a struct with data populated successfully
	// ...
	type returnVals struct {
		// the key will be the name of struct field unless you give it an explicit JSON tag
		Body  string `json:"body"`
		Error string `json:"error"`
		Valid bool   `json:"valid"`
	}

	//conditional respo body?
	respBody := returnVals{
		//temp, burde være samme body
		Body:  "This is an opinion I need to share with the world",
		Error: "nothing to see here",
		Valid: false,
	}

	//generic error
	dat, err := json.Marshal(respBody)
	if err != nil {
		//log.Printf("Error marshalling JSON: %s", err)

		respBody.Error = "\"error\": \"Something went wrong\""

		errdat, err2 := json.Marshal(respBody.Error)
		if err2 != nil {
			//second error, cant seem to send stuff
			return

		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(500)
		writer.Write(errdat)
		return

		//respBody.Error = "\"error\": \"Something went wrong\""
	}

	respBody.Valid = true

	dat3, err := json.Marshal(respBody.Valid)
	if err != nil {
		//log.Printf("Error marshalling JSON: %s", err)
		return

		//respBody.Error = "\"error\": \"Something went wrong\""
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(400)
	writer.Write(dat3)

	//return early? or let this be?

	writer.Header().Set("Content-Type", "application/json")
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
