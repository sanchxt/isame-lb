package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	// Health check endpoint
	http.HandleFunc("/health", healthHandler)

	// Basic info endpoint
	http.HandleFunc("/", rootHandler)

	addr := ":8080"
	log.Printf("Isame Load Balancer starting on %s", addr)

	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"service":"isame-lb","version":"0.1.0","phase":"0"}`)
}
