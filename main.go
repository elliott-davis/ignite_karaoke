package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed templates
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

func main() {
	giphyAPIKey := os.Getenv("GIPHY_API_KEY")
	if giphyAPIKey == "" {
		fmt.Println("Warning: GIPHY_API_KEY not set. Falling back to placeholder GIF.")
	}
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	if googleAPIKey == "" {
		log.Fatal("GOOGLE_API_KEY not set")
	}

	templates := template.Must(template.ParseFS(templateFS, "templates/*.html"))

	generator, err := NewAiGenerator(googleAPIKey)
	if err != nil {
		log.Fatalf("failed to create AI generator: %v", err)
	}

	app := &App{
		templates:    templates,
		giphyAPIKey:  giphyAPIKey,
		googleAPIKey: googleAPIKey,
		generator:    generator,
		usedGifs:     make(map[string]bool),
	}

	http.HandleFunc("/", app.indexHandler)
	http.HandleFunc("/admin", app.adminHandler)
	http.HandleFunc("/participants", app.participantsHandler)
	http.HandleFunc("/remove-participant", app.removeParticipantHandler)
	http.HandleFunc("/next-participant", app.nextParticipantHandler)
	http.HandleFunc("/game/", app.gameHandler)
	http.HandleFunc("/api/game-data/", app.gameDataHandler)

	// Serve static files from embedded filesystem
	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
