package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (apiCtx *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.path == "/reset" {
			apiCtx.fileserverHits.Store(0)
		} else {
			apiCtx.fileserverHits.Add(1)
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealthzHTTPRequests(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func handleMetricsAndResetHTTPRequest(w http.ResponseWriter, req *http.Request) {
	fmt.Sprintf("Hits: %v", apiContext.fileserverHits.Load())
	if req.URL.path == "/metrics" {
		w.Write([]byte(fmt.Sprintf("Hits: %v", apiContext.fileserverHits.Load())))
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
	mux.HandleFunc("/healthz", handleHealthzHTTPRequests)

	mux.HandleFunc("/metrics", handleMetricsAndResetHTTPRequest);
	mux.HandleFunc("/reset", apiContext.middlewareMetricsInc(handleMetricsAndResetHTTPRequest));

	fileServerHandler := http.FileServer(http.Dir("."))
	mux.Handle("/app/", apiContext.middlewareMetricsInc(http.StripPrefix("/app", fileServerHandler)))

	srv := &http.Server{
		Addr:    ":" + portToUse,
		Handler: mux,
	}

	
	log.Printf("Serving on port: %s\n", portToUse)
	log.Fatal(srv.ListenAndServe())
}