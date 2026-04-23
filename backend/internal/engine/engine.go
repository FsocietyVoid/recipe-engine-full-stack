package engine

import (
	"sort"
	"strings"

	"recipe-scraper/internal/models"
)

// MatchResult holds a recipe with its match score for a given ingredient query
type MatchResult struct {
	Recipe        models.Recipe
	MatchedCount  int     // how many queried ingredients were matched
	TotalRequired int     // total ingredients in the recipe
	Score         float64 // 0.0–1.0 percentage of recipe ingredients matched
}

// Engine provides ingredient-based recipe suggestions
type Engine struct {
	recipes []models.Recipe
}

// New creates an Engine loaded with the provided recipes
func New(recipes []models.Recipe) *Engine {
	return &Engine{recipes: recipes}
}

// SuggestByIngredients returns recipes ranked by how many of the provided
// ingredients they use. availableIngredients should be a list of ingredient
// names (e.g. ["chicken", "yogurt", "garlic"]).
//
// Options:
//   minMatchCount – only return recipes where at least N ingredients match (default 1)
//   maxResults    – cap results (0 = no cap)
func (e *Engine) SuggestByIngredients(available []string, minMatchCount, maxResults int) []MatchResult {
	if minMatchCount < 1 {
		minMatchCount = 1
	}
	normalised := make([]string, len(available))
	for i, a := range available {
		normalised[i] = strings.ToLower(strings.TrimSpace(a))
	}

	var results []MatchResult

	for _, recipe := range e.recipes {
		matched := countMatches(recipe, normalised)
		if matched < minMatchCount {
			continue
		}
		total := len(recipe.Ingredients)
		if total == 0 {
			total = len(recipe.MainIngredients)
		}
		score := 0.0
		if total > 0 {
			score = float64(matched) / float64(total)
		}
		results = append(results, MatchResult{
			Recipe:        recipe,
			MatchedCount:  matched,
			TotalRequired: total,
			Score:         score,
		})
	}

	// Sort by matched count desc, then score desc
	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchedCount != results[j].MatchedCount {
			return results[i].MatchedCount > results[j].MatchedCount
		}
		return results[i].Score > results[j].Score
	})

	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}
	return results
}

// SearchByTitle returns recipes whose title contains all of the given keywords
func (e *Engine) SearchByTitle(keywords ...string) []models.Recipe {
	var out []models.Recipe
	for _, r := range e.recipes {
		titleLC := strings.ToLower(r.Title)
		ok := true
		for _, kw := range keywords {
			if !strings.Contains(titleLC, strings.ToLower(kw)) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, r)
		}
	}
	return out
}

// FilterByCuisine returns recipes matching the given cuisine (case-insensitive)
func (e *Engine) FilterByCuisine(cuisine string) []models.Recipe {
	lc := strings.ToLower(cuisine)
	var out []models.Recipe
	for _, r := range e.recipes {
		if strings.Contains(strings.ToLower(r.Cuisine), lc) {
			out = append(out, r)
		}
	}
	return out
}

// FilterByDiet returns veg or non-veg recipes
func (e *Engine) FilterByDiet(veg bool) []models.Recipe {
	want := "non vegetarian"
	if veg {
		want = "vegetarian"
	}
	var out []models.Recipe
	for _, r := range e.recipes {
		if strings.Contains(strings.ToLower(r.DietType), want) ||
			(veg && !strings.Contains(strings.ToLower(r.DietType), "non")) {
			out = append(out, r)
		}
	}
	return out
}

// AllIngredients returns a de-duplicated list of all ingredient names in the
// dataset – useful for building an autocomplete index.
func (e *Engine) AllIngredients() []string {
	seen := make(map[string]bool)
	var out []string
	for _, r := range e.recipes {
		for _, ing := range r.Ingredients {
			n := strings.ToLower(strings.TrimSpace(ing.Name))
			if n != "" && !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
		for _, mi := range r.MainIngredients {
			n := strings.ToLower(strings.TrimSpace(mi))
			if n != "" && !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}

// ── helpers ────────────────────────────────────────────────────────────────────

func countMatches(r models.Recipe, available []string) int {
	count := 0
	// Build a combined ingredient list from both detailed + main ingredients
	allNames := make([]string, 0, len(r.Ingredients)+len(r.MainIngredients))
	for _, ing := range r.Ingredients {
		allNames = append(allNames, strings.ToLower(ing.Name))
	}
	for _, mi := range r.MainIngredients {
		allNames = append(allNames, strings.ToLower(mi))
	}

	for _, avail := range available {
		for _, name := range allNames {
			if name == "" {
				continue
			}
			// substring match to handle "kashmiri red chilli powder" matching "chilli"
			if strings.Contains(name, avail) || strings.Contains(avail, name) {
				count++
				break
			}
		}
	}
	return count
}
