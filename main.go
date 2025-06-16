package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"
)

var (
	participants     []string
	participantsMu   sync.Mutex
	templates        *template.Template
	usedGifs         = make(map[string]bool)
	usedGifsMu       sync.Mutex
	giphyAPIKey      string
	giphyCache       []string
	giphyCacheMu     sync.Mutex
	giphyCacheExpiry time.Time
)

func main() {
	giphyAPIKey = os.Getenv("GIPHY_API_KEY")
	if giphyAPIKey == "" {
		fmt.Println("Warning: GIPHY_API_KEY not set. Falling back to placeholder GIF.")
	}
	// Parse templates
	templates = template.Must(template.ParseGlob("templates/*.html"))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/admin", adminHandler)
	http.HandleFunc("/participants", participantsHandler)
	http.HandleFunc("/remove-participant", removeParticipantHandler)
	http.HandleFunc("/next-participant", nextParticipantHandler)
	http.HandleFunc("/game/", gameHandler)
	http.HandleFunc("/api/game-data/", gameDataHandler)

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	participantsMu.Lock()
	defer participantsMu.Unlock()

	nextParticipant := ""
	if len(participants) > 0 {
		nextParticipant = participants[0]
	}

	data := struct {
		Participants []string
		Next         string
	}{
		Participants: participants,
		Next:         nextParticipant,
	}

	templates.ExecuteTemplate(w, "index.html", data)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	participantsMu.Lock()
	defer participantsMu.Unlock()
	data := struct {
		Participants []string
	}{
		Participants: participants,
	}
	templates.ExecuteTemplate(w, "admin.html", data)
}

func participantsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	names := r.FormValue("names")
	newParticipants := strings.Split(names, "\n")

	participantsMu.Lock()
	defer participantsMu.Unlock()

	participants = []string{}
	for _, name := range newParticipants {
		name = strings.TrimSpace(name)
		if name != "" {
			participants = append(participants, name)
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func removeParticipantHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nameToRemove := r.FormValue("name")
	if nameToRemove == "" {
		http.Error(w, "Participant name cannot be empty", http.StatusBadRequest)
		return
	}

	participantsMu.Lock()
	defer participantsMu.Unlock()

	var newParticipants []string
	for _, p := range participants {
		if p != nameToRemove {
			newParticipants = append(newParticipants, p)
		}
	}
	participants = newParticipants

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func nextParticipantHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	participantsMu.Lock()
	defer participantsMu.Unlock()

	if len(participants) > 0 {
		participants = participants[1:]
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func generateBusinessIdea(ctx context.Context) (string, string, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return "", "", fmt.Errorf("GOOGLE_API_KEY not set")
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return "", "", err
	}
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr[float32](0.9),
	}

	prompt := genai.Text("Generate a fake, humorous business name and a slogan for it. Return it as 'Name: <name> Slogan: <slogan>'")
	resp, err := client.Models.GenerateContent(ctx, "gemini-1.5-pro-latest", prompt, config)

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

func generateImagePrompt(ctx context.Context) (string, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GOOGLE_API_KEY not set")
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return "", err
	}

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

	resp, err := client.Models.GenerateContent(ctx, "gemini-1.5-pro-latest", prompt, config)
	if err != nil {
		return "", err
	}

	return resp.Text(), nil
}

func generateImage(ctx context.Context, prompt string) (string, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GOOGLE_API_KEY not set")
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return "", err
	}

	config := &genai.GenerateImagesConfig{
		NumberOfImages: 1,
	}

	response, err := client.Models.GenerateImages(
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

func getClappingGiphy(ctx context.Context) (string, error) {
	if giphyAPIKey == "" {
		return "https://media.giphy.com/media/3o7abB06u9bNzA8lu8/giphy.gif", nil
	}

	giphyCacheMu.Lock()
	if time.Now().Before(giphyCacheExpiry) && len(giphyCache) > 0 {
		gif := getRandomGifFromCache()
		giphyCacheMu.Unlock()
		return gif, nil
	}

	// Invalidate cache and fetch new gifs
	giphyCache = []string{}
	giphyCacheMu.Unlock()

	url := fmt.Sprintf("https://api.giphy.com/v1/gifs/search?api_key=%s&q=clapping&limit=50&rating=g", giphyAPIKey)
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

	giphyCacheMu.Lock()
	defer giphyCacheMu.Unlock()
	for _, gif := range giphyResp.Data {
		giphyCache = append(giphyCache, gif.Images.Original.URL)
	}
	giphyCacheExpiry = time.Now().Add(1 * time.Hour) // Cache for 1 hour

	return getRandomGifFromCache(), nil
}

func getRandomGifFromCache() string {
	// Assumes giphyCacheMu is already locked
	if len(giphyCache) == 0 {
		return "https://media.giphy.com/media/3o7abB06u9bNzA8lu8/giphy.gif" // Fallback
	}

	for i := 0; i < 10; i++ { // Try 10 times to find an unused gif
		gif := giphyCache[rand.Intn(len(giphyCache))]
		usedGifsMu.Lock()
		if !usedGifs[gif] {
			usedGifs[gif] = true
			usedGifsMu.Unlock()
			return gif
		}
		usedGifsMu.Unlock()
	}

	// If we can't find an unused one, just return a random one
	return giphyCache[rand.Intn(len(giphyCache))]
}

func gameHandler(w http.ResponseWriter, r *http.Request) {
	participantName := strings.TrimPrefix(r.URL.Path, "/game/")

	data := struct {
		ParticipantName string
	}{
		ParticipantName: participantName,
	}

	templates.ExecuteTemplate(w, "game.html", data)
}

func gameDataHandler(w http.ResponseWriter, r *http.Request) {
	participantName := strings.TrimPrefix(r.URL.Path, "/api/game-data/")

	businessName, slogan, err := generateBusinessIdea(r.Context())
	if err != nil {
		http.Error(w, "Failed to generate business idea", http.StatusInternalServerError)
		return
	}

	clappingGif, err := getClappingGiphy(r.Context())
	if err != nil {
		fmt.Printf("Failed to get clapping gif: %s\n", err)
		clappingGif = "https://media.giphy.com/media/3o7abB06u9bNzA8lu8/giphy.gif"
	}

	imagePrompt1, err := generateImagePrompt(r.Context())
	if err != nil {
		log.Printf("Failed to generate image prompt 1: %v", err)
		http.Error(w, "Failed to generate image prompt 1", http.StatusInternalServerError)
		return
	}
	log.Printf("Generated Image Prompt 1: %s", imagePrompt1)
	image1, err := generateImage(r.Context(), imagePrompt1)
	if err != nil {
		log.Printf("Failed to generate image 1 with prompt '%s': %v", imagePrompt1, err)
		http.Error(w, "Failed to generate image 1", http.StatusInternalServerError)
		return
	}

	imagePrompt2, err := generateImagePrompt(r.Context())
	if err != nil {
		log.Printf("Failed to generate image prompt 2: %v", err)
		http.Error(w, "Failed to generate image prompt 2", http.StatusInternalServerError)
		return
	}
	log.Printf("Generated Image Prompt 2: %s", imagePrompt2)
	image2, err := generateImage(r.Context(), imagePrompt2)
	if err != nil {
		log.Printf("Failed to generate image 2 with prompt '%s': %v", imagePrompt2, err)
		http.Error(w, "Failed to generate image 2", http.StatusInternalServerError)
		return
	}

	data := struct {
		ParticipantName string `json:"participantName"`
		BusinessName    string `json:"businessName"`
		Slogan          string `json:"slogan"`
		Image1          string `json:"image1"`
		Image2          string `json:"image2"`
		ClappingGif     string `json:"clappingGif"`
	}{
		ParticipantName: participantName,
		BusinessName:    businessName,
		Slogan:          slogan,
		Image1:          image1,
		Image2:          image2,
		ClappingGif:     clappingGif,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
