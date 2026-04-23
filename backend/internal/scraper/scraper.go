package scraper

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/queue"

	"recipe-scraper/internal/models"
)

const (
	baseURL       = "https://www.sanjeevkapoor.com"
	recipeListURL = "https://www.sanjeevkapoor.com/Recipe"
	rssFeedURL    = "https://www.sanjeevkapoor.com/rss/categories/Recipe"
)

// Config holds all scraper tuning parameters
type Config struct {
	Parallelism      int
	RequestDelay     time.Duration
	RandomDelay      time.Duration
	MaxListingPages  int // max number of paginated listing pages to scrape (0 = skip pagination)
	MaxRecipes       int
	FollowPagination bool
	OutputDir        string
	Logger           *log.Logger
}

// DefaultConfig returns a polite default configuration
func DefaultConfig() Config {
	return Config{
		Parallelism:      1,
		RequestDelay:     3 * time.Second,
		RandomDelay:      2 * time.Second,
		MaxListingPages:  0,
		MaxRecipes:       0,
		FollowPagination: true,
		OutputDir:        "./output",
	}
}

// Scraper orchestrates the crawl
type Scraper struct {
	cfg       Config
	logger    *log.Logger
	listings  []models.RecipeListing
	recipes   []models.Recipe
	stats     models.ScrapeStats
	onRecipe  func(models.Recipe)
	onListing func(models.RecipeListing)
}

// New creates a new Scraper with the given config
func New(cfg Config) *Scraper {
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}
	return &Scraper{
		cfg:    cfg,
		logger: logger,
	}
}

// OnRecipe registers a callback invoked whenever a recipe is fully scraped
func (s *Scraper) OnRecipe(fn func(models.Recipe)) {
	s.onRecipe = fn
}

// OnListing registers a callback for each listing card found
func (s *Scraper) OnListing(fn func(models.RecipeListing)) {
	s.onListing = fn
}

// Run starts the scraping process and blocks until done
func (s *Scraper) Run() ([]models.Recipe, models.ScrapeStats, error) {
	s.stats.StartedAt = time.Now()

	// Shared cookie jar for colly
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, s.stats, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Helper: add realistic browser headers for colly
	addHeaders := func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Referer", "https://www.sanjeevkapoor.com/")
		r.Headers.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
		r.Headers.Set("Sec-Ch-Ua-Mobile", "?0")
		r.Headers.Set("Sec-Ch-Ua-Platform", `"Windows"`)
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "same-origin")
		r.Headers.Set("Sec-Fetch-User", "?1")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
	}

	// Recipe collector (colly) – for detail pages only
	recipeCol := colly.NewCollector(
		colly.AllowedDomains("www.sanjeevkapoor.com"),
		colly.Async(true),
	)
	recipeCol.SetCookieJar(jar)
	extensions.RandomUserAgent(recipeCol)
	recipeCol.OnRequest(addHeaders)

	recipeCol.Limit(&colly.LimitRule{
		DomainGlob:  "*sanjeevkapoor.com*",
		Parallelism: s.cfg.Parallelism,
		Delay:       s.cfg.RequestDelay,
		RandomDelay: s.cfg.RandomDelay,
	})

	// Parse each recipe page
	recipeCol.OnHTML("html", func(e *colly.HTMLElement) {
		recipe, err := parseRecipePage(e)
		if err != nil {
			s.logger.Printf("[recipe] error parsing %s: %v", e.Request.URL, err)
			s.stats.TotalErrors++
			return
		}
		s.recipes = append(s.recipes, recipe)
		s.stats.TotalScraped++
		s.logger.Printf("[recipe] ✓ scraped %q (%d ingredients)", recipe.Title, len(recipe.Ingredients))
		if s.onRecipe != nil {
			s.onRecipe(recipe)
		}
	})

	recipeCol.OnError(func(r *colly.Response, err error) {
		s.logger.Printf("[recipe] HTTP %d on %s: %v", r.StatusCode, r.Request.URL, err)
		s.stats.TotalErrors++
	})

	// ── Step 1: Get recipe URLs from RSS feed (fast, latest recipes) ──
	allRecipeURLs := make(map[string]bool)
	s.logger.Printf("[scraper] fetching recipe URLs from RSS feed: %s", rssFeedURL)
	rssURLs, err := s.getRecipeURLsFromRSS()
	if err != nil {
		s.logger.Printf("[scraper] RSS feed failed: %v – continuing with pagination only", err)
	} else {
		s.logger.Printf("[scraper] RSS feed gave %d URLs", len(rssURLs))
		for _, u := range rssURLs {
			allRecipeURLs[u] = true
		}
	}

	// ── Step 2: If pagination enabled, scrape listing pages for more URLs ──
	if s.cfg.FollowPagination && s.cfg.MaxListingPages > 0 {
		s.logger.Printf("[scraper] starting paginated listing scrape (max pages = %d)", s.cfg.MaxListingPages)
		ctx, cancel := chromedp.NewContext(context.Background())
		defer cancel()
		ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		for page := 1; page <= s.cfg.MaxListingPages; page++ {
			if s.cfg.MaxRecipes > 0 && len(allRecipeURLs) >= s.cfg.MaxRecipes {
				break
			}
			pageURL := fmt.Sprintf("%s?page=%d", recipeListURL, page)
			s.logger.Printf("[scraper] fetching listing page %d: %s", page, pageURL)
			links, err := s.getRecipeLinksWithChromedp(ctx, pageURL)
			if err != nil {
				s.logger.Printf("[scraper] page %d error: %v – stopping pagination", page, err)
				break
			}
			newCount := 0
			for _, link := range links {
				if s.cfg.MaxRecipes > 0 && len(allRecipeURLs) >= s.cfg.MaxRecipes {
					break
				}
				if !allRecipeURLs[link] {
					allRecipeURLs[link] = true
					newCount++
				}
			}
			s.logger.Printf("[scraper] page %d gave %d new URLs (total: %d)", page, newCount, len(allRecipeURLs))
			time.Sleep(s.cfg.RequestDelay)
		}
	}

	s.logger.Printf("[scraper] total unique recipe URLs to scrape: %d", len(allRecipeURLs))
	if len(allRecipeURLs) == 0 {
		return nil, s.stats, fmt.Errorf("no recipe URLs found")
	}

	// Limit by MaxRecipes
	urls := make([]string, 0, len(allRecipeURLs))
	for u := range allRecipeURLs {
		urls = append(urls, u)
	}
	if s.cfg.MaxRecipes > 0 && len(urls) > s.cfg.MaxRecipes {
		urls = urls[:s.cfg.MaxRecipes]
	}

	// Add URLs to colly queue
	q, _ := queue.New(s.cfg.Parallelism, &queue.InMemoryQueueStorage{MaxSize: 100000})
	for _, u := range urls {
		q.AddURL(u)
	}

	// Run the colly queue
	if err := q.Run(recipeCol); err != nil {
		return nil, s.stats, fmt.Errorf("queue run failed: %w", err)
	}
	recipeCol.Wait()

	s.stats.CompletedAt = time.Now()
	s.logger.Printf("[scraper] done: scraped=%d errors=%d duration=%s",
		s.stats.TotalScraped, s.stats.TotalErrors,
		s.stats.CompletedAt.Sub(s.stats.StartedAt).Round(time.Second))

	return s.recipes, s.stats, nil
}

// getRecipeURLsFromRSS fetches the RSS feed and extracts all recipe URLs
func (s *Scraper) getRecipeURLsFromRSS() ([]string, error) {
	resp, err := http.Get(rssFeedURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var feed struct {
		Channel struct {
			Items []struct {
				Link string `xml:"link"`
			} `xml:"item"`
		} `xml:"channel"`
	}

	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("XML parse: %w", err)
	}

	var urls []string
	seen := make(map[string]bool)
	for _, item := range feed.Channel.Items {
		link := strings.TrimSpace(item.Link)
		if link == "" || !strings.Contains(link, "/Recipe/") {
			continue
		}
		if !seen[link] {
			seen[link] = true
			urls = append(urls, link)
		}
	}
	return urls, nil
}

// getRecipeLinksWithChromedp loads a listing page and extracts all recipe URLs.
func (s *Scraper) getRecipeLinksWithChromedp(ctx context.Context, pageURL string) ([]string, error) {
	var links []string
	err := chromedp.Run(ctx,
		chromedp.Navigate(pageURL),
		chromedp.Sleep(3*time.Second), // wait for JS to load
		chromedp.Evaluate(`
			(() => {
				const anchors = Array.from(document.querySelectorAll('a[href*="/Recipe/"]'));
				return anchors.map(a => a.href);
			})();
		`, &links),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp evaluate error: %w", err)
	}

	// Deduplicate and validate
	seen := make(map[string]bool)
	var valid []string
	for _, link := range links {
		if !strings.Contains(link, "/Recipe/") {
			continue
		}
		if !seen[link] {
			seen[link] = true
			valid = append(valid, link)
		}
	}
	return valid, nil
}

// parseRecipePage extracts data from a single recipe HTML page (colly callback)
func parseRecipePage(e *colly.HTMLElement) (models.Recipe, error) {
	r := models.Recipe{
		URL:       e.Request.URL.String(),
		ScrapedAt: time.Now(),
	}

	// ID from last part of URL
	parts := strings.Split(strings.TrimRight(r.URL, "/"), "/")
	if len(parts) > 0 {
		r.ID = parts[len(parts)-1]
	}

	// Title
	r.Title = strings.TrimSpace(e.ChildText("h1.primary_font"))

	// Description (first paragraph in article summary)
	r.Description = strings.TrimSpace(e.ChildText("section.article-summary p"))

	// Image
	r.ImageURL = e.ChildAttr("article img", "src")
	if r.ImageURL == "" {
		r.ImageURL = e.ChildAttr(".article-cover img", "src")
	}

	// Author & date
	r.Author = strings.TrimSpace(e.ChildText(".art-author a"))
	r.PublishedAt = strings.TrimSpace(e.ChildText("time"))

	// Metadata table
	e.DOM.Find("figure.tinymce-table-div table tr").Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() != 2 {
			return
		}
		key := normalise(tds.Eq(0).Text())
		val := strings.TrimSpace(tds.Eq(1).Text())
		switch key {
		case "mainingredients":
			r.MainIngredients = splitComma(val)
		case "cuisine":
			r.Cuisine = val
		case "course":
			r.Course = val
		case "preptime":
			r.PrepTime = val
		case "cooktime":
			r.CookTime = val
		case "serve":
			r.Servings = val
		case "taste":
			r.Taste = val
		case "levelofcooking":
			r.CookingLevel = val
		case "others":
			r.DietType = val
		}
	})

	// Ingredients section
	e.DOM.Find("#postContent h2, #post-container h2").Each(func(_ int, h2 *goquery.Selection) {
		if strings.Contains(strings.ToLower(h2.Text()), "ingredients") {
			ul := h2.NextUntil("h2").Filter("ul").First()
			ul.Find("li").Each(func(_ int, li *goquery.Selection) {
				raw := strings.TrimSpace(li.Text())
				if raw != "" {
					r.Ingredients = append(r.Ingredients, parseIngredient(raw))
				}
			})
		}
	})

	// Method / Steps
	e.DOM.Find("#postContent h2, #post-container h2").Each(func(_ int, h2 *goquery.Selection) {
		if strings.Contains(strings.ToLower(h2.Text()), "method") {
			ol := h2.NextUntil("h2").Filter("ol").First()
			ol.Find("li").Each(func(_ int, li *goquery.Selection) {
				step := strings.TrimSpace(li.Text())
				if step != "" {
					r.Steps = append(r.Steps, step)
				}
			})
		}
	})

	// Tags
	e.DOM.Find(".tags a").Each(func(_ int, a *goquery.Selection) {
		tag := strings.TrimSpace(a.Text())
		if tag != "" {
			tag = strings.TrimPrefix(tag, "#")
			r.Tags = append(r.Tags, tag)
		}
	})

	if r.Title == "" {
		return r, fmt.Errorf("no title found – not a recipe page")
	}
	return r, nil
}

// parseIngredient extracts quantity, unit, and name from a raw string
var quantityRe = regexp.MustCompile(`^(\d[\d/\s½¼¾⅓⅔⅛⅜⅝⅞]*)\s*`)
var unitWords = map[string]bool{
	"teaspoon": true, "teaspoons": true, "tsp": true,
	"tablespoon": true, "tablespoons": true, "tbsp": true,
	"cup": true, "cups": true,
	"gram": true, "grams": true, "g": true,
	"kilogram": true, "kilograms": true, "kg": true,
	"ml": true, "milliliter": true, "millilitre": true,
	"liter": true, "litre": true, "l": true,
	"piece": true, "pieces": true,
	"pinch": true, "handful": true,
}

func parseIngredient(raw string) models.Ingredient {
	ing := models.Ingredient{Raw: raw}
	rest := raw

	if m := quantityRe.FindStringSubmatch(rest); len(m) >= 2 {
		ing.Quantity = strings.TrimSpace(m[1])
		rest = strings.TrimSpace(rest[len(m[0]):])
	}

	words := strings.Fields(rest)
	if len(words) > 0 && unitWords[strings.ToLower(words[0])] {
		ing.Unit = words[0]
		rest = strings.TrimSpace(strings.Join(words[1:], " "))
	}

	name := regexp.MustCompile(`\(.*?\)`).ReplaceAllString(rest, "")
	name = strings.TrimRight(strings.TrimSpace(name), ",")
	ing.Name = name

	return ing
}

// ── Helpers ─────────────────────────────────────────────────────────────

func normalise(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}