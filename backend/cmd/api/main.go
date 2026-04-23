package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"github.com/joho/godotenv"

	"recipe-scraper/internal/engine"
	"recipe-scraper/internal/storage"
)

func main() {

	if err := godotenv.Load(); err != nil {
    log.Println("No .env file found, using system env")
}
	// Load recipes from output directory
	store, err := storage.New("./output")
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	recipes, err := store.LoadAll()
	if err != nil {
		log.Fatal("Failed to load recipes:", err)
	}

	log.Printf("Loaded %d recipes", len(recipes))
	eng := engine.New(recipes)

	// Enable CORS for frontend
	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next(w, r)
		}
	}

	// Endpoint: /suggest?ingredients=chicken,garlic,yogurt&minMatch=2&top=10
	http.HandleFunc("/suggest", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		ingredientsStr := query.Get("ingredients")
		if ingredientsStr == "" {
			http.Error(w, "missing 'ingredients' parameter", http.StatusBadRequest)
			return
		}

		ingredients := strings.Split(ingredientsStr, ",")
		for i := range ingredients {
			ingredients[i] = strings.TrimSpace(ingredients[i])
		}

		minMatch := 1
		if mm := query.Get("minMatch"); mm != "" {
			if v, err := strconv.Atoi(mm); err == nil && v > 0 {
				minMatch = v
			}
		}

		top := 10
		if t := query.Get("top"); t != "" {
			if v, err := strconv.Atoi(t); err == nil && v > 0 {
				top = v
			}
		}

		results := eng.SuggestByIngredients(ingredients, minMatch, top)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(results); err != nil {
			log.Printf("JSON encode error: %v", err)
		}
	}))

	// Endpoint: /recipes (optional: list all recipes)
	http.HandleFunc("/recipes", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipes)
	}))

	// Endpoint: /match (POST) – for compatibility with frontend
	http.HandleFunc("/match", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Ingredients []string `json:"ingredients"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		results := eng.SuggestByIngredients(req.Ingredients, 1, 20)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}))

	http.HandleFunc("/ai/tips", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "POST required", http.StatusMethodNotAllowed)
        return
    }
    apiKey := os.Getenv("GEMINI_API_KEY")
    if apiKey == "" {
        http.Error(w, "GEMINI_API_KEY not set", http.StatusInternalServerError)
        return
    }
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    // Forward to Gemini API
    url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", apiKey)
    req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()
    respBody, _ := io.ReadAll(resp.Body)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(resp.StatusCode)
    w.Write(respBody)
	}))

	port := ":8080"
	log.Printf("API server listening on http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}