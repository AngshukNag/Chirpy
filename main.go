package main

import (
	"fmt"
	"log"
	"net/http"
)

func handleHealthzHTTPRequests(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main() {
	const port = "8082"

	fmt.Println("--- RUNNING ABSOLUTE LOCAL VERSION ---")
	mux := http.NewServeMux()
	// mux.HandleFunc("/healthz", handleHealthzHTTPRequests)

	fileServerHandler := http.FileServer(http.Dir("."))
	// mux.Handle("/app", http.StripPrefix("/app", fileServerHandler))
	mux.Handle("/", fileServerHandler)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	
	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}