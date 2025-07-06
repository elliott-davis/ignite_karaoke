package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"google.golang.org/genai"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
}

// DefaultRetryConfig provides sensible defaults for retry behavior
var DefaultRetryConfig = RetryConfig{
	MaxRetries: 5,
	BaseDelay:  time.Second,
}

// retryWithBackoff executes a function with exponential backoff retry logic
func retryWithBackoff(ctx context.Context, config RetryConfig, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			multiplier := 1 << uint(attempt-1)
			delay := time.Duration(float64(config.BaseDelay) * float64(multiplier))
			jitter := time.Duration(rand.Float64() * float64(delay) * 0.1) // 10% jitter
			totalDelay := delay + jitter

			log.Printf("Retry attempt %d/%d after %v", attempt, config.MaxRetries, totalDelay)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(totalDelay):
			}
		}

		if err := operation(); err != nil {
			lastErr = err
			log.Printf("Operation failed on attempt %d: %v", attempt+1, err)
			continue
		}

		if attempt > 0 {
			log.Printf("Operation succeeded on retry attempt %d", attempt)
		}
		return nil
	}

	return fmt.Errorf("operation failed after %d attempts, last error: %w", config.MaxRetries+1, lastErr)
}

type Generator interface {
	GenerateBusinessIdea(ctx context.Context) (string, string, error)
	GenerateImagePrompt(ctx context.Context) (string, error)
	GenerateImage(ctx context.Context, prompt string) (string, error)
}

type GiphyClient interface {
	GetClappingGiphy(ctx context.Context) (string, error)
}

type AiGenerator struct {
	googleAPIKey string
	client       *genai.Client
}

func NewAiGenerator(apiKey string) (*AiGenerator, error) {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}
	return &AiGenerator{
		googleAPIKey: apiKey,
		client:       client,
	}, nil
}

func (g *AiGenerator) GenerateBusinessIdea(ctx context.Context) (string, string, error) {
	businessTypes := []string{"a mobile app", "a subscription box", "a gourmet food truck", "a line of smart home devices", "a bespoke tailoring service", "a virtual reality arcade", "an artisanal coffee shop", "a pet psychic agency", "a zero-gravity yoga studio"}
	targetAudiences := []string{"time-traveling tourists", "sentient houseplants", "retired superheroes", "aliens on vacation", "ghosts with unfinished business", "zombies who are into personal growth", "dolphins who want to be web developers", "cats who are learning to code", "very-online vampires"}
	absurdProblems := []string{"socks that are always lonely", "pigeons that are too loud", "a toaster with an attitude problem", "the existential dread of a Roomba", "lost TV remotes", "dreams that are too boring", "awkward silences in elevators", "when your pet starts talking about philosophy", "running out of things to watch on streaming services"}

	request := BusinessIdeaRequest{
		BusinessType:   getRandomElement(businessTypes),
		TargetAudience: getRandomElement(targetAudiences),
		AbsurdProblem:  getRandomElement(absurdProblems),
		Instructions:   "Generate a fake, humorous business name and a slogan for it based on the fields above. Return it as 'Name: <name> Slogan: <slogan>'",
	}

	jsonRequest, err := json.Marshal(request)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal business idea request: %w", err)
	}

	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr[float32](0.9),
	}

	finalPrompt := fmt.Sprintf("Based on the following JSON, fulfill the instructions:\n\n%s", string(jsonRequest))
	prompt := genai.Text(finalPrompt)

	var resp *genai.GenerateContentResponse
	err = retryWithBackoff(ctx, DefaultRetryConfig, func() error {
		var apiErr error
		resp, apiErr = g.client.Models.GenerateContent(ctx, "gemini-1.5-pro-latest", prompt, config)
		return apiErr
	})

	if err != nil {
		return "", "", fmt.Errorf("failed to generate business idea after retries: %w", err)
	}

	var textContent string
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.Text != "" {
				textContent = string(part.Text)
				break
			}
		}
	}

	if textContent == "" {
		return "", "", fmt.Errorf("failed to extract text from Gemini response")
	}

	parts := strings.Split(textContent, "Slogan:")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected response format from Gemini: %s", textContent)
	}
	namePart := strings.TrimSpace(strings.TrimPrefix(parts[0], "Name:"))
	sloganPart := strings.TrimSpace(parts[1])

	return namePart, sloganPart, nil
}

type BusinessIdeaRequest struct {
	BusinessType   string `json:"business_type"`
	TargetAudience string `json:"target_audience"`
	AbsurdProblem  string `json:"absurd_problem"`
	Instructions   string `json:"instructions"`
}

type ImagePromptRequest struct {
	CharacterAgeRange string `json:"character_age_range"`
	Setting           string `json:"setting"`
	AbsurdTwist       string `json:"absurd_twist"`
	VisualStyle       string `json:"visual_style"`
	FinalPrompt       string `json:"final_prompt"`
}

func getRandomElement(slice []string) string {
	return slice[rand.Intn(len(slice))]
}

func (g *AiGenerator) GenerateImagePrompt(ctx context.Context) (string, error) {
	characterAges := []string{"child", "teenager", "adult", "middle-aged", "elderly"}
	settings := []string{"unexpected public place", "outer space", "underwater", "historic era", "corporate office", "dreamlike zone"}
	absurdTwists := []string{"prop or situation that contradicts logic or expectations", "a mundane task performed in an extreme environment", "animals behaving like humans in a specific, detailed way", "a historical figure using modern technology", "an inanimate object coming to life with a strong personality"}

	request := ImagePromptRequest{
		CharacterAgeRange: getRandomElement(characterAges),
		Setting:           getRandomElement(settings),
		AbsurdTwist:       getRandomElement(absurdTwists),
		VisualStyle:       "photorealistic",
		FinalPrompt:       "[Write a single, richly detailed, photorealistic image prompt for a SFW AI image generator. It should use these fields to describe a vivid, absurd and comedic scene. The description must be specific, visual, and funny — like something from a dream or a comedy sketch. Avoid clichés, generic phrasing and jokes involving suicide.]",
	}

	jsonRequest, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal prompt request: %w", err)
	}

	config := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr[float32](0.9),
		MaxOutputTokens: 300,
	}

	finalPrompt := fmt.Sprintf("Based on the following JSON, generate the 'final_prompt':\n\n%s", string(jsonRequest))
	prompt := genai.Text(finalPrompt)

	var resp *genai.GenerateContentResponse
	err = retryWithBackoff(ctx, DefaultRetryConfig, func() error {
		var apiErr error
		resp, apiErr = g.client.Models.GenerateContent(ctx, "gemini-1.5-pro-latest", prompt, config)
		return apiErr
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate image prompt after retries: %w", err)
	}

	return resp.Text(), nil
}

func (g *AiGenerator) GenerateImage(ctx context.Context, prompt string) (string, error) {
	config := &genai.GenerateImagesConfig{
		NumberOfImages: 1,
	}

	var response *genai.GenerateImagesResponse
	err := retryWithBackoff(ctx, DefaultRetryConfig, func() error {
		var apiErr error
		response, apiErr = g.client.Models.GenerateImages(
			ctx,
			"imagen-3.0-generate-002",
			"generate an image of: "+prompt,
			config,
		)
		return apiErr
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate image after retries: %w", err)
	}

	for _, image := range response.GeneratedImages {
		return fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(image.Image.ImageBytes)), nil
	}

	return "", fmt.Errorf("no image data in response from prompt: %s", prompt)
}

func (app *App) GetClappingGiphy(ctx context.Context) (string, error) {
	if app.giphyAPIKey == "" {
		return "https://media.giphy.com/media/3o7abB06u9bNzA8lu8/giphy.gif", nil
	}

	app.giphyCacheMu.Lock()
	if time.Now().Before(app.giphyCacheExpiry) && len(app.giphyCache) > 0 {
		gif := app.getRandomGifFromCache()
		app.giphyCacheMu.Unlock()
		return gif, nil
	}

	// Invalidate cache and fetch new gifs
	app.giphyCache = []string{}
	app.giphyCacheMu.Unlock()

	url := fmt.Sprintf("https://api.giphy.com/v1/gifs/search?api_key=%s&q=clapping&limit=50&rating=g", app.giphyAPIKey)

	var giphyResp struct {
		Data []struct {
			Images struct {
				Original struct {
					URL string `json:"url"`
				} `json:"original"`
			} `json:"images"`
		} `json:"data"`
	}

	err := retryWithBackoff(ctx, DefaultRetryConfig, func() error {
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to make HTTP request to Giphy API: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Giphy API returned status code %d", resp.StatusCode)
		}

		if err := json.NewDecoder(resp.Body).Decode(&giphyResp); err != nil {
			return fmt.Errorf("failed to decode Giphy API response: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Failed to fetch from Giphy API after retries: %v", err)
		return "https://media.giphy.com/media/3o7abB06u9bNzA8lu8/giphy.gif", nil
	}

	app.giphyCacheMu.Lock()
	defer app.giphyCacheMu.Unlock()
	for _, gif := range giphyResp.Data {
		app.giphyCache = append(app.giphyCache, gif.Images.Original.URL)
	}
	// Shuffle the cache to ensure variety
	rand.Shuffle(len(app.giphyCache), func(i, j int) {
		app.giphyCache[i], app.giphyCache[j] = app.giphyCache[j], app.giphyCache[i]
	})
	app.usedGifs = make(map[string]bool) // Reset used gifs when we refresh the cache

	app.giphyCacheExpiry = time.Now().Add(1 * time.Hour) // Cache for 1 hour

	return app.getRandomGifFromCache(), nil
}

func (app *App) getRandomGifFromCache() string {
	// Assumes giphyCacheMu is already locked
	if len(app.giphyCache) == 0 {
		return "https://media.giphy.com/media/3o7abB06u9bNzA8lu8/giphy.gif" // Fallback
	}

	app.usedGifsMu.Lock()
	defer app.usedGifsMu.Unlock()
	for _, gif := range app.giphyCache {
		if !app.usedGifs[gif] {
			app.usedGifs[gif] = true
			return gif
		}
	}

	// If all gifs from the cache have been used, reset the used map and return a random one
	log.Println("All cached Giphy GIFs have been used. Resetting and serving a random one.")
	app.usedGifs = make(map[string]bool)
	gif := app.giphyCache[rand.Intn(len(app.giphyCache))]
	app.usedGifs[gif] = true
	return gif
}

// generateGameContent creates a complete GameContent with all required assets
func (app *App) generateGameContent(ctx context.Context) (*GameContent, error) {
	// Generate business idea
	businessName, slogan, err := app.generator.GenerateBusinessIdea(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate business idea: %w", err)
	}

	// Generate clapping GIF
	clappingGif, err := app.GetClappingGiphy(ctx)
	if err != nil {
		log.Printf("Failed to get clapping gif: %v", err)
		clappingGif = "https://media.giphy.com/media/3o7abB06u9bNzA8lu8/giphy.gif"
	}

	// Generate first image
	imagePrompt1, err := app.generator.GenerateImagePrompt(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image prompt 1: %w", err)
	}

	image1, err := app.generator.GenerateImage(ctx, imagePrompt1)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image 1: %w", err)
	}

	// Generate second image
	imagePrompt2, err := app.generator.GenerateImagePrompt(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image prompt 2: %w", err)
	}

	image2, err := app.generator.GenerateImage(ctx, imagePrompt2)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image 2: %w", err)
	}

	return &GameContent{
		BusinessName: businessName,
		Slogan:       slogan,
		Image1:       image1,
		Image2:       image2,
		ClappingGif:  clappingGif,
		CreatedAt:    time.Now(),
	}, nil
}

// StartContentPreloader starts the background content preloader
func (app *App) StartContentPreloader(ctx context.Context) {
	app.preloadMu.Lock()
	defer app.preloadMu.Unlock()

	if app.preloadRunning {
		return // Already running
	}

	app.preloadRunning = true
	app.preloadStop = make(chan struct{})

	go app.runContentPreloader(ctx)
}

// StopContentPreloader stops the background content preloader
func (app *App) StopContentPreloader() {
	app.preloadMu.Lock()
	defer app.preloadMu.Unlock()

	if !app.preloadRunning {
		return // Not running
	}

	close(app.preloadStop)
	app.preloadRunning = false
}

// isPreloadRunning returns whether the content preloader is currently running
func (app *App) isPreloadRunning() bool {
	app.preloadMu.Lock()
	defer app.preloadMu.Unlock()
	return app.preloadRunning
}

// runContentPreloader is the main preloader loop
func (app *App) runContentPreloader(ctx context.Context) {
	log.Println("Starting content preloader...")

	// Initial load - fill cache to 80% capacity
	initialTarget := int(float64(app.contentCache.maxSize) * 0.8)
	for i := 0; i < initialTarget; i++ {
		select {
		case <-app.preloadStop:
			log.Println("Content preloader stopped during initial load")
			return
		case <-ctx.Done():
			log.Println("Content preloader stopped due to context cancellation")
			return
		default:
		}

		content, err := app.generateGameContent(ctx)
		if err != nil {
			log.Printf("Failed to generate content during initial load: %v", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}

		app.contentCache.Push(*content)
		log.Printf("Initial preload: generated content %d/%d", i+1, initialTarget)
	}

	app.contentCache.SetLoaded()
	log.Printf("Initial content preload completed. Cache size: %d", app.contentCache.Size())

	// Maintenance loop - keep cache topped up
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-app.preloadStop:
			log.Println("Content preloader stopped")
			return
		case <-ctx.Done():
			log.Println("Content preloader stopped due to context cancellation")
			return
		case <-ticker.C:
			// Check if cache needs refilling
			cacheSize := app.contentCache.Size()
			targetSize := int(float64(app.contentCache.maxSize) * 0.8)

			if cacheSize < targetSize {
				log.Printf("Cache low (%d/%d), generating new content...", cacheSize, targetSize)

				content, err := app.generateGameContent(ctx)
				if err != nil {
					log.Printf("Failed to generate content during maintenance: %v", err)
					continue
				}

				app.contentCache.Push(*content)
				log.Printf("Generated new content. Cache size: %d", app.contentCache.Size())
			}
		}
	}
}
