<div align="center">

# Recipe Engine

**A full-stack, ingredient-driven recipe discovery platform with AI-powered cooking assistance.**

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react)](https://react.dev/)
[![TypeScript](https://img.shields.io/badge/TypeScript-6.0-3178C6?style=flat-square&logo=typescript)](https://www.typescriptlang.org/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker)](https://docs.docker.com/compose/)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

</div>

---

## Overview

Recipe Engine is a containerised full-stack application that matches available ingredients to the most suitable recipes from a structured dataset, scored by ingredient coverage. An integrated **Google Gemini 1.5 Flash** assistant surfaces contextual cooking tips for any selected recipe.

The project ships two independent executables:

| Binary | Description |
|---|---|
| `cmd/api` | HTTP REST API server (port `8080`) |
| `cmd/scraper` | CLI tool for scraping, querying, and inspecting recipe data |

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Docker Network                   │
│                                                     │
│  ┌──────────────────┐       ┌──────────────────┐   │
│  │   Frontend       │──────▶│   Backend API    │   │
│  │  React + Nginx   │ :80   │   Go net/http    │   │
│  │  TypeScript/Vite │       │   Port :8080     │   │
│  └──────────────────┘       └────────┬─────────┘   │
│                                      │              │
│                              ┌───────┴──────┐       │
│                              │  output/     │       │
│                              │  *.json      │       │
│                              │  (recipes)   │       │
│                              └──────────────┘       │
└─────────────────────────────────────────────────────┘
                                      │
                               ┌──────▼──────┐
                               │ Gemini API  │
                               │ (external)  │
                               └─────────────┘
```

---

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | React 19, TypeScript 6, Vite 8, Tailwind CSS 4 |
| Backend | Go 1.26, `net/http` (stdlib only) |
| Web Scraper | Colly v2, Chromedp, goquery |
| AI Integration | Google Gemini 1.5 Flash (REST proxy) |
| Web Server | Nginx (production frontend) |
| Containerisation | Docker, Docker Compose |

---

## Project Structure

```
recipe-engine-full-stack/
├── backend/
│   ├── cmd/
│   │   ├── api/main.go              # REST API server
│   │   └── scraper/main.go          # CLI scraper & query tool
│   ├── internal/
│   │   ├── engine/engine.go         # Ingredient matching & scoring
│   │   ├── models/recipe.go         # Domain models
│   │   ├── scraper/scraper.go       # Web scraper (Colly + Chromedp)
│   │   └── storage/storage.go       # JSON file persistence
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── AISuggestions.tsx
│   │   │   ├── RecipeCard.tsx
│   │   │   ├── RecipeDetails.tsx
│   │   │   ├── RecipeModal.tsx
│   │   │   └── SearchBox.tsx
│   │   ├── services/                # Axios API client
│   │   ├── types.ts
│   │   └── App.tsx
│   ├── Dockerfile
│   └── package.json
└── docker-compose.yml
```

---

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) & Docker Compose
- [Go 1.21+](https://go.dev/dl/) *(for running the scraper locally)*
- A [Google Gemini API key](https://aistudio.google.com/app/apikey)

---

### 1. Clone the Repository

```bash
git clone https://github.com/FsocietyVoid/recipe-engine-full-stack.git
cd recipe-engine-full-stack
```

---

### 2. Populate the Recipe Dataset

The API server reads recipe data from `backend/output/`. This directory must be populated before the application can serve results. Run the scraper locally:

```bash
cd backend
go run ./cmd/scraper \
  -mode=scrape \
  -output=./output \
  -max=200 \
  -parallel=2 \
  -delay=2s
```

> **Note:** Use `-max=50` for an initial test run. Recipes are saved incrementally, so the process can be safely interrupted and resumed.

---

### 3. Configure Environment Variables

Create a `.env` file in the `backend/` directory:

```env
GEMINI_API_KEY=your_api_key_here
```

---

### 4. Build and Run

```bash
# From the project root
docker compose up --build
```

| Service | Address |
|---|---|
| Frontend | http://localhost |
| Backend API | http://localhost:8080 |

---

## API Reference

### `GET /suggest`

Returns a ranked list of recipes that match the provided ingredients.

**Query Parameters**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `ingredients` | `string` | *required* | Comma-separated ingredient names |
| `minMatch` | `int` | `1` | Minimum number of required matches |
| `top` | `int` | `10` | Maximum number of results |

**Example Request**
```http
GET /suggest?ingredients=chicken,garlic,yogurt&minMatch=2&top=5
```

**Example Response**
```json
[
  {
    "Recipe": {
      "id": "murgh-makhani",
      "title": "Murgh Makhani",
      "cuisine": "Indian",
      "diet_type": "Non Vegetarian",
      "prep_time": "30 minutes",
      "cook_time": "45 minutes"
    },
    "MatchedCount": 3,
    "TotalRequired": 12,
    "Score": 0.25
  }
]
```

---

### `POST /match`

Accepts a JSON body as an alternative to query string parameters.

**Request Body**
```json
{
  "ingredients": ["chicken", "garlic", "yogurt"]
}
```

---

### `GET /recipes`

Returns the complete list of all loaded recipes.

---

### `POST /ai/tips`

Proxies a request to the Gemini 1.5 Flash API and returns AI-generated cooking tips for a given recipe. Requires `GEMINI_API_KEY` to be configured.

---

## CLI Reference

The `cmd/scraper` binary supports the following modes:

```bash
# Scrape recipes from the web and save to disk
go run ./cmd/scraper -mode=scrape -output=./output -max=100

# Query recipes by ingredient without starting the server
go run ./cmd/scraper -mode=suggest -ingredients="paneer,tomato,cream" -min-match=2 -top=5

# List all unique ingredients present in the dataset
go run ./cmd/scraper -mode=list-ingredients -output=./output

# Dump the full recipe dataset as formatted JSON to stdout
go run ./cmd/scraper -mode=dump -output=./output
```

**Available Flags**

| Flag | Default | Description |
|---|---|---|
| `-mode` | `scrape` | Execution mode: `scrape` \| `suggest` \| `list-ingredients` \| `dump` |
| `-output` | `./output` | Directory used to save and load recipe JSON files |
| `-max` | `50` | Maximum recipes to scrape (`0` = unlimited) |
| `-pages` | `0` | Maximum listing pages to crawl (`0` = unlimited) |
| `-parallel` | `2` | Number of concurrent recipe fetch workers |
| `-delay` | `2s` | Delay between outbound requests |
| `-ingredients` | — | Comma-separated ingredients *(suggest mode only)* |
| `-min-match` | `2` | Minimum ingredient match threshold *(suggest mode only)* |
| `-top` | `10` | Maximum results to return *(suggest mode only)* |

---

## Matching Algorithm

Each candidate recipe is evaluated against the supplied ingredient list using the following scoring model:

1. **Match Count** — the number of user-supplied ingredients found in the recipe (case-insensitive substring match across both `Ingredients` and `MainIngredients` fields).
2. **Coverage Score** — `matched_count / total_recipe_ingredients`, normalised to the range `[0.0, 1.0]`.

Results are sorted **by match count descending**, with coverage score used as a tiebreaker. Only recipes meeting the `minMatch` threshold are included.

---

## Data Model

```go
// Recipe represents a fully structured recipe record.
type Recipe struct {
    ID              string       `json:"id"`
    Title           string       `json:"title"`
    URL             string       `json:"url"`
    Description     string       `json:"description"`
    ImageURL        string       `json:"image_url"`
    Author          string       `json:"author"`
    Cuisine         string       `json:"cuisine"`
    Course          string       `json:"course"`
    PrepTime        string       `json:"prep_time"`
    CookTime        string       `json:"cook_time"`
    Servings        string       `json:"servings"`
    CookingLevel    string       `json:"cooking_level"`
    DietType        string       `json:"diet_type"`   // "Vegetarian" | "Non Vegetarian"
    MainIngredients []string     `json:"main_ingredients"`
    Ingredients     []Ingredient `json:"ingredients"`
    Steps           []string     `json:"steps"`
    Tags            []string     `json:"tags"`
    ScrapedAt       time.Time    `json:"scraped_at"`
}

// Ingredient represents a single parsed ingredient entry.
type Ingredient struct {
    Raw      string `json:"raw"`       // e.g. "2½ tablespoons yogurt"
    Quantity string `json:"quantity"`  // e.g. "2½"
    Unit     string `json:"unit"`      // e.g. "tablespoons"
    Name     string `json:"name"`      // e.g. "yogurt"
}
```

---

## Contributing

Contributions are welcome. Please open an issue to discuss proposed changes before submitting a pull request. Ensure all code is formatted with `gofmt` (backend) and passes `eslint` (frontend) before submission.

---

## License

This project is licensed under the terms described in the [LICENSE](LICENSE) file.
