package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	fmt.PrintLn("Starting HTTP Server ......")
	server.ListenAndServe()
}