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
	resp, err := g.client.Models.GenerateContent(ctx, "gemini-1.5-pro-latest", prompt, config)

	if err != nil {
		return "", "", err
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
		MaxOutputTokens: 400,
	}

	finalPrompt := fmt.Sprintf("Based on the following JSON, generate the 'final_prompt':\n\n%s", string(jsonRequest))

	prompt := genai.Text(finalPrompt)

	resp, err := g.client.Models.GenerateContent(ctx, "gemini-1.5-pro-latest", prompt, config)
	if err != nil {
		return "", err
	}

	return resp.Text(), nil
}

func (g *AiGenerator) GenerateImage(ctx context.Context, prompt string) (string, error) {
	config := &genai.GenerateImagesConfig{
		NumberOfImages: 1,
	}

	response, err := g.client.Models.GenerateImages(
		ctx,
		"imagen-3.0-generate-002",
		"generate an image of: "+prompt,
		config,
	)

	if err != nil {
		return "", err
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
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var giphyResp struct {
		Data []struct {
			Images struct {
				Original struct {
					URL string `json:"url"`
				} `json:"original"`
			} `json:"images"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&giphyResp); err != nil {
		return "", err
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
