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
	"time"

	"github.com/joho/godotenv"

	"recipe-scraper/internal/engine"
	"recipe-scraper/internal/storage"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

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

	ragURL := os.Getenv("RAG_SERVICE_URL") // e.g. http://rag-service:8090

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

	// ── /suggest ─────────────────────────────────────────────────────────────
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

	// ── /recipes ──────────────────────────────────────────────────────────────
	http.HandleFunc("/recipes", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipes)
	}))

	// ── /match ────────────────────────────────────────────────────────────────
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

	// ── /ai/tips ──────────────────────────────────────────────────────────────
	http.HandleFunc("/ai/tips", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// ── Try RAG service first ─────────────────────────────────────────────
		if ragURL != "" {
			ragResp, err := callRAGService(ragURL, body)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-AI-Source", "local-rag") // useful for frontend debugging
				w.Write(ragResp)
				return
			}
			log.Printf("RAG service unavailable, falling back to Gemini: %v", err)
		}

		// ── Fallback: Gemini ──────────────────────────────────────────────────
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			http.Error(w, "RAG service unavailable and GEMINI_API_KEY not set", http.StatusInternalServerError)
			return
		}

		geminiURL := fmt.Sprintf(
			"https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s",
			apiKey,
		)
		geminiReq, _ := http.NewRequest("POST", geminiURL, bytes.NewReader(body))
		geminiReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(geminiReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-AI-Source", "gemini")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
	}))

	// ── /ai/source ── lets the frontend check which AI backend is active ──────
	http.HandleFunc("/ai/source", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		source := "gemini"
		if ragURL != "" {
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Get(ragURL + "/health")
			if err == nil && resp.StatusCode == http.StatusOK {
				source = "local-rag"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"source": source})
	}))

	port := ":8080"
	log.Printf("API server listening on http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

// callRAGService translates the incoming request body into the RAG service
// format, calls it, and returns the response as a JSON blob the frontend
// already understands (same shape as the Gemini response).
func callRAGService(ragURL string, body []byte) ([]byte, error) {
	// Parse what the frontend sent (Gemini format)
	var geminiReq map[string]interface{}
	if err := json.Unmarshal(body, &geminiReq); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	// Extract the prompt text buried in Gemini's nested structure:
	// { "contents": [{ "parts": [{ "text": "..." }] }] }
	promptText := extractGeminiPrompt(geminiReq)

	// Build the RAG service payload
	ragPayload, err := json.Marshal(map[string]interface{}{
		"recipe_title": extractField(promptText, "recipe"),
		"ingredients":  extractIngredients(promptText),
		"question":     promptText,
	})
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 60 * time.Second} // LLM can be slow
	resp, err := client.Post(ragURL+"/rag/tips", "application/json", bytes.NewReader(ragPayload))
	if err != nil {
		return nil, fmt.Errorf("rag service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rag service returned %d", resp.StatusCode)
	}

	ragBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse RAG response: { "tips": "..." }
	var ragResp map[string]string
	if err := json.Unmarshal(ragBody, &ragResp); err != nil {
		return nil, fmt.Errorf("invalid rag response: %w", err)
	}

	// Wrap in Gemini-compatible shape so the frontend needs zero changes
	wrapped, err := json.Marshal(map[string]interface{}{
		"candidates": []map[string]interface{}{
			{
				"content": map[string]interface{}{
					"parts": []map[string]string{
						{"text": ragResp["tips"]},
					},
				},
			},
		},
	})
	return wrapped, err
}

// extractGeminiPrompt pulls the text out of Gemini's nested request format.
func extractGeminiPrompt(req map[string]interface{}) string {
	contents, ok := req["contents"].([]interface{})
	if !ok || len(contents) == 0 {
		return ""
	}
	first, ok := contents[0].(map[string]interface{})
	if !ok {
		return ""
	}
	parts, ok := first["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return ""
	}
	part, ok := parts[0].(map[string]interface{})
	if !ok {
		return ""
	}
	text, _ := part["text"].(string)
	return text
}

// extractField does a simple keyword search in the prompt for a recipe name.
func extractField(prompt, _ string) string {
	lower := strings.ToLower(prompt)
	for _, kw := range []string{"recipe for ", "cooking ", "make ", "preparing "} {
		if idx := strings.Index(lower, kw); idx != -1 {
			rest := prompt[idx+len(kw):]
			if end := strings.IndexAny(rest, ".,\n?"); end != -1 {
				return strings.TrimSpace(rest[:end])
			}
			if len(rest) < 60 {
				return strings.TrimSpace(rest)
			}
		}
	}
	return "recipe"
}

// extractIngredients pulls comma/newline separated ingredients from the prompt.
func extractIngredients(prompt string) []string {
	var ingredients []string
	for _, line := range strings.Split(prompt, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "ingredient") {
			continue
		}
		for _, part := range strings.Split(line, ",") {
			part = strings.TrimSpace(part)
			if part != "" && len(part) < 40 {
				ingredients = append(ingredients, part)
			}
		}
	}
	return ingredients
}