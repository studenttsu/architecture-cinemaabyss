package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                  string
	MonolithURL           string
	MoviesServiceURL      string
	EventsServiceURL      string
	GradualMigration      bool
	MoviesMigrationPercent int
}

var config Config

func main() {
	rand.Seed(time.Now().UnixNano())
	
	loadConfig()
	
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", proxyHandler)
	
	port := config.Port
	if port == "" {
		port = "8000"
	}
	
	log.Printf("Starting API Gateway on port %s", port)
	log.Printf("Monolith URL: %s", config.MonolithURL)
	log.Printf("Movies Service URL: %s", config.MoviesServiceURL)
	log.Printf("Events Service URL: %s", config.EventsServiceURL)
	log.Printf("Gradual Migration: %v", config.GradualMigration)
	log.Printf("Movies Migration Percent: %d%%", config.MoviesMigrationPercent)
	
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func loadConfig() {
	config = Config{
		Port:                  os.Getenv("PORT"),
		MonolithURL:           os.Getenv("MONOLITH_URL"),
		MoviesServiceURL:      os.Getenv("MOVIES_SERVICE_URL"),
		EventsServiceURL:      os.Getenv("EVENTS_SERVICE_URL"),
		GradualMigration:      os.Getenv("GRADUAL_MIGRATION") == "true",
		MoviesMigrationPercent: 0,
	}
	
	if percent := os.Getenv("MOVIES_MIGRATION_PERCENT"); percent != "" {
		if p, err := strconv.Atoi(percent); err == nil {
			config.MoviesMigrationPercent = p
		}
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	
	var targetURL string
	
	if strings.HasPrefix(path, "/api/movies") {
		targetURL = routeMoviesRequest()
	} else if strings.HasPrefix(path, "/api/events") {
		targetURL = config.EventsServiceURL
	} else if strings.HasPrefix(path, "/api/users") || 
	          strings.HasPrefix(path, "/api/payments") || 
	          strings.HasPrefix(path, "/api/subscriptions") {
		targetURL = config.MonolithURL
	} else {
		targetURL = config.MonolithURL
	}
	
	if targetURL == "" {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	
	proxyRequest(w, r, targetURL)
}

func routeMoviesRequest() string {
	if !config.GradualMigration {
		return config.MonolithURL
	}
	
	randomValue := rand.Intn(100)
	
	if randomValue < config.MoviesMigrationPercent {
		log.Printf("Routing to Movies Microservice (migration: %d%%)", config.MoviesMigrationPercent)
		return config.MoviesServiceURL
	}
	
	log.Printf("Routing to Monolith (migration: %d%%)", config.MoviesMigrationPercent)
	return config.MonolithURL
}

func proxyRequest(w http.ResponseWriter, r *http.Request, targetURL string) {
	fullURL := targetURL + r.URL.Path
	if r.URL.RawQuery != "" {
		fullURL += "?" + r.URL.RawQuery
	}
	
	log.Printf("Proxying %s %s to %s", r.Method, r.URL.Path, fullURL)
	
	proxyReq, err := http.NewRequest(r.Method, fullURL, r.Body)
	if err != nil {
		log.Printf("Error creating proxy request: %v", err)
		http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
		return
	}
	
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("Error forwarding request: %v", err)
		http.Error(w, fmt.Sprintf("Error forwarding request: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	
	w.WriteHeader(resp.StatusCode)
	
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response: %v", err)
	}
}
