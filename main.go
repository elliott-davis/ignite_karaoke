package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
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

	// Configure cache size from environment variable
	cacheSize := 20 // Default cache size for production
	if cacheSizeStr := os.Getenv("CACHE_SIZE"); cacheSizeStr != "" {
		if size, err := strconv.Atoi(cacheSizeStr); err == nil && size > 0 {
			cacheSize = size
		} else {
			log.Printf("Invalid CACHE_SIZE value '%s', using default: %d", cacheSizeStr, cacheSize)
		}
	}

	// Configure whether to enable preloading
	enablePreload := true // Default enabled for production
	if preloadStr := os.Getenv("ENABLE_PRELOAD"); preloadStr != "" {
		if preload, err := strconv.ParseBool(preloadStr); err == nil {
			enablePreload = preload
		} else {
			log.Printf("Invalid ENABLE_PRELOAD value '%s', using default: %t", preloadStr, enablePreload)
		}
	}

	log.Printf("Content cache configured: size=%d, preload=%t", cacheSize, enablePreload)

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
		contentCache: NewContentCache(cacheSize),
	}

	// Start background content preloader only if enabled
	if enablePreload {
		app.StartContentPreloader(context.Background())
		log.Println("Content preloader started")
	} else {
		log.Println("Content preloader disabled")
	}

	http.HandleFunc("/", app.indexHandler)
	http.HandleFunc("/admin", app.adminHandler)
	http.HandleFunc("/participants", app.participantsHandler)
	http.HandleFunc("/remove-participant", app.removeParticipantHandler)
	http.HandleFunc("/next-participant", app.nextParticipantHandler)
	http.HandleFunc("/preload-cache", app.preloadCacheHandler)
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
