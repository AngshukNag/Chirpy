package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"encoding/json"
	"strings"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

type Chirp struct {
	Body string `json:"body"`
}

type CleanedChirp struct {
	Body string `json:"cleaned_body"`
}

type ChirpyError struct {
	Error string `json:"error"`
}

type ValidSuccess struct {
	Valid bool `json:"valid"`
}

func (apiCtx *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/admin/reset" {
			apiCtx.fileserverHits.Store(0)
		} else {
			apiCtx.fileserverHits.Add(1)
		}
		next.ServeHTTP(w, req)
	})
}

func handleHealthzHTTPRequests(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func handleValidateChirpHTTPRequests(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	responseBody := Chirp{}
	err := decoder.Decode(&responseBody)
	if err != nil {
		errorVal := ChirpyError{
			Error: "Something went wrong",
		}
		data, err := json.Marshal(errorVal)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write(data)
		return
	}

	if len(responseBody.Body) > 140 {
		chirpError := ChirpyError{
			Error: "Chirp is too long",
		}

		data, err := json.Marshal(chirpError)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		w.Write(data)
	} else {
		cleaned_body := removeProfaneWords(responseBody.Body)
		fmt.Println("Cleaned Chrip: ", cleaned_body)
		cleanedChripMessage := CleanedChirp{
			Body: cleaned_body,
		}

		data, err := json.Marshal(cleanedChripMessage)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(data)
	}
}

func removeProfaneWords(input string) string {
	profaneWords := map[string]struct{} {
			"kerfuffle": struct{}{},
			"sharbert": struct{}{},
			"fornax": struct{}{},
		}

	words := strings.Fields(input)

	for index, word := range words {
		if _, ok := profaneWords[strings.ToLower(word)]; ok == true {
			words[index] = "****"
		}
	}

	return strings.Join(words, " ")
}

func handleMetricsAndResetHTTPRequest(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/admin/metrics" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, apiContext.fileserverHits.Load())))
	}
}

var apiContext *apiConfig

// func init() {
// 	apiContext = &apiConfig{}
// }

func main() {
	const testPort = "8082"
	const livePort = "8080"

	var portToUse string
	apiContext = &apiConfig{}

	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "live" {
			portToUse = livePort
		} else {
			portToUse = testPort
		}
	} else {
		portToUse = testPort
	}

	fmt.Println("--- RUNNING ABSOLUTE LOCAL VERSION !!!!! ---")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handleHealthzHTTPRequests)

	mux.HandleFunc("GET /admin/metrics", handleMetricsAndResetHTTPRequest);
	mux.HandleFunc("POST /api/validate_chirp", handleValidateChirpHTTPRequests);
	mux.Handle("POST /admin/reset", apiContext.middlewareMetricsInc(http.HandlerFunc(handleMetricsAndResetHTTPRequest)));

	fileServerHandler := http.FileServer(http.Dir("."))
	mux.Handle("/app/", apiContext.middlewareMetricsInc(http.StripPrefix("/app", fileServerHandler)))

	srv := &http.Server{
		Addr:    ":" + portToUse,
		Handler: mux,
	}

	
	log.Printf("Serving on port: %s\n", portToUse)
	log.Fatal(srv.ListenAndServe())
}