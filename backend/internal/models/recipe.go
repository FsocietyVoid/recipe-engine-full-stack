package models

import "time"

// Recipe represents a fully scraped recipe from sanjeevkapoor.com
type Recipe struct {
	ID              string       `json:"id"`
	Title           string       `json:"title"`
	URL             string       `json:"url"`
	Description     string       `json:"description"`
	ImageURL        string       `json:"image_url"`
	Author          string       `json:"author"`
	PublishedAt     string       `json:"published_at"`
	Cuisine         string       `json:"cuisine"`
	Course          string       `json:"course"`
	PrepTime        string       `json:"prep_time"`
	CookTime        string       `json:"cook_time"`
	Servings        string       `json:"servings"`
	Taste           string       `json:"taste"`
	CookingLevel    string       `json:"cooking_level"`
	DietType        string       `json:"diet_type"` // Veg / Non-Veg
	MainIngredients []string     `json:"main_ingredients"`
	Ingredients     []Ingredient `json:"ingredients"`
	Steps           []string     `json:"steps"`
	Tags            []string     `json:"tags"`
	ScrapedAt       time.Time    `json:"scraped_at"`
}

// Ingredient represents a single recipe ingredient with quantity
type Ingredient struct {
	Raw      string `json:"raw"`       // full original text e.g. "2½ tablespoons yogurt"
	Quantity string `json:"quantity"`  // e.g. "2½"
	Unit     string `json:"unit"`      // e.g. "tablespoons"
	Name     string `json:"name"`      // normalised ingredient name e.g. "yogurt"
}

// RecipeListing is the lightweight card data scraped from the listing page
type RecipeListing struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
	Author      string `json:"author"`
	PublishedAt string `json:"published_at"`
}

// ScrapeStats holds counters for a scraping run
type ScrapeStats struct {
	TotalVisited  int
	TotalScraped  int
	TotalErrors   int
	TotalSkipped  int
	StartedAt     time.Time
	CompletedAt   time.Time
}
