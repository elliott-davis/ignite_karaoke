package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
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
	http.HandleFunc("/game/", gameHandler)

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
	templates.ExecuteTemplate(w, "admin.html", nil)
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

func generateBusinessIdea(ctx context.Context) (string, string, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return "", "", fmt.Errorf("GOOGLE_API_KEY not set")
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", "", err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	prompt := genai.Text("Generate a fake, humorous business name and a slogan for it. Return it as 'Name: <name> Slogan: <slogan>'")

	resp, err := model.GenerateContent(ctx, prompt)
	if err != nil {
		return "", "", err
	}

	// It's a bit more complex to get the exact text, so we'll have to iterate.
	var textContent string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if txt, ok := part.(genai.Text); ok {
					textContent = string(txt)
					break
				}
			}
		}
		if textContent != "" {
			break
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

	data := struct {
		ParticipantName string
		BusinessName    string
		Slogan          string
		Image1          string
		Image2          string
		ClappingGif     string
	}{
		ParticipantName: participantName,
		BusinessName:    businessName,
		Slogan:          slogan,
		Image1:          fmt.Sprintf("https://picsum.photos/seed/%d/800/600", rand.Intn(1000)),
		Image2:          fmt.Sprintf("https://picsum.photos/seed/%d/800/600", rand.Intn(1000)+1000),
		ClappingGif:     clappingGif,
	}

	templates.ExecuteTemplate(w, "game.html", data)
}
