package main

import (
	"log"
	"net/http"
)

func main() {
	const port = "8082"

	mux := http.NewServeMux()
	fileServerHandler := http.FileServer(http.Dir("."))

	mux.Handle("/", fileServerHandler)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}