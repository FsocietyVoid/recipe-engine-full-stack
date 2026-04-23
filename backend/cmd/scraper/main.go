package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"recipe-scraper/internal/engine"
	"recipe-scraper/internal/models"
	"recipe-scraper/internal/scraper"
	"recipe-scraper/internal/storage"
)

func main() {
	// ── Flags ──────────────────────────────────────────────────────────────────
	mode := flag.String("mode", "scrape", "Mode: scrape | suggest | list-ingredients")
	outputDir := flag.String("output", "./output", "Directory to save / load recipe JSON files")
	maxRecipes := flag.Int("max", 50, "Max recipes to scrape (0 = unlimited)")
	maxPages := flag.Int("pages", 0, "Max listing pages to crawl (0 = unlimited)")
	parallelism := flag.Int("parallel", 2, "Number of parallel recipe requests")
	delay := flag.Duration("delay", 2*time.Second, "Delay between requests")
	ingredients := flag.String("ingredients", "", "Comma-separated ingredients for suggest mode")
	minMatch := flag.Int("min-match", 2, "Minimum ingredient matches for suggest mode")
	topN := flag.Int("top", 10, "Max results in suggest mode")
	flag.Parse()

	logger := log.New(os.Stdout, "[recipe-engine] ", log.LstdFlags)

	store, err := storage.New(*outputDir)
	if err != nil {
		logger.Fatalf("storage init: %v", err)
	}

	switch *mode {

	// ── SCRAPE ─────────────────────────────────────────────────────────────────
	case "scrape":
		cfg := scraper.Config{
			Parallelism:      *parallelism,
			RequestDelay:     *delay,
			RandomDelay:      time.Second,
			MaxRecipes:       *maxRecipes,
			MaxListingPages:  *maxPages,
			FollowPagination: true,
			OutputDir:        *outputDir,
			Logger:           logger,
		}

		s := scraper.New(cfg)

		// Save each recipe incrementally as it arrives
		s.OnRecipe(func(r models.Recipe) {
			if err := store.SaveRecipe(r); err != nil {
				logger.Printf("save error for %s: %v", r.ID, err)
			}
		})

		recipes, stats, err := s.Run()
		if err != nil {
			logger.Fatalf("scrape failed: %v", err)
		}

		// Also write the combined file
		if err := store.SaveAll(recipes); err != nil {
			logger.Printf("save all: %v", err)
		}

		fmt.Printf("\n✅  Scraping complete\n")
		fmt.Printf("   Scraped  : %d recipes\n", stats.TotalScraped)
		fmt.Printf("   Errors   : %d\n", stats.TotalErrors)
		fmt.Printf("   Duration : %s\n", stats.CompletedAt.Sub(stats.StartedAt).Round(time.Second))
		fmt.Printf("   Output   : %s\n", *outputDir)

	// ── SUGGEST ────────────────────────────────────────────────────────────────
	case "suggest":
		if *ingredients == "" {
			logger.Fatal("provide -ingredients flag, e.g. -ingredients=\"chicken,yogurt,garlic\"")
		}

		recipes, err := store.LoadAll()
		if err != nil {
			logger.Fatalf("load recipes: %v", err)
		}

		eng := engine.New(recipes)
		ingList := splitTrimmed(*ingredients, ",")

		fmt.Printf("\n🔍  Finding recipes that use: %s\n\n", strings.Join(ingList, ", "))
		results := eng.SuggestByIngredients(ingList, *minMatch, *topN)

		if len(results) == 0 {
			fmt.Println("No matching recipes found. Try reducing -min-match or adding more ingredients.")
			return
		}

		for i, res := range results {
			fmt.Printf("%2d. %s\n", i+1, res.Recipe.Title)
			fmt.Printf("    Matched: %d/%d ingredients (%.0f%%)\n",
				res.MatchedCount, res.TotalRequired, res.Score*100)
			fmt.Printf("    Cuisine: %s | Course: %s | Diet: %s\n",
				res.Recipe.Cuisine, res.Recipe.Course, res.Recipe.DietType)
			fmt.Printf("    URL    : %s\n\n", res.Recipe.URL)
		}

	// ── LIST-INGREDIENTS ───────────────────────────────────────────────────────
	case "list-ingredients":
		recipes, err := store.LoadAll()
		if err != nil {
			logger.Fatalf("load recipes: %v", err)
		}
		eng := engine.New(recipes)
		all := eng.AllIngredients()
		fmt.Printf("Found %d unique ingredients:\n\n", len(all))
		for _, ing := range all {
			fmt.Println(" •", ing)
		}

	// ── DUMP (debug) ───────────────────────────────────────────────────────────
	case "dump":
		recipes, err := store.LoadAll()
		if err != nil {
			logger.Fatalf("load recipes: %v", err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(recipes) //nolint

	default:
		logger.Fatalf("unknown mode %q — choose: scrape | suggest | list-ingredients", *mode)
	}
}

func splitTrimmed(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
