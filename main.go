package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func handleHealthzHTTPRequests(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main() {
	const testPort = "8082"
	const livePort = "8080"

	var portToUse string

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

	fileServerHandler := http.FileServer(http.Dir("."))
	mux.Handle("/app/", http.StripPrefix("/app", fileServerHandler))

	srv := &http.Server{
		Addr:    ":" + portToUse,
		Handler: mux,
	}

	
	log.Printf("Serving on port: %s\n", portToUse)
	log.Fatal(srv.ListenAndServe())
}