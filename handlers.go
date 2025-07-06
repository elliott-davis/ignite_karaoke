package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

func (app *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	app.participantsMu.Lock()
	defer app.participantsMu.Unlock()

	nextParticipant := ""
	if len(app.participants) > 0 {
		nextParticipant = app.participants[0]
	}

	data := struct {
		Participants []string
		Next         string
	}{
		Participants: app.participants,
		Next:         nextParticipant,
	}

	app.templates.ExecuteTemplate(w, "index.html", data)
}

func (app *App) adminHandler(w http.ResponseWriter, r *http.Request) {
	app.participantsMu.Lock()
	defer app.participantsMu.Unlock()

	data := struct {
		Participants   []string
		CacheSize      int
		CacheLoaded    bool
		MaxCacheSize   int
		PreloadRunning bool
	}{
		Participants:   app.participants,
		CacheSize:      app.contentCache.Size(),
		CacheLoaded:    app.contentCache.IsLoaded(),
		MaxCacheSize:   app.contentCache.maxSize,
		PreloadRunning: app.isPreloadRunning(),
	}
	app.templates.ExecuteTemplate(w, "admin.html", data)
}

func (app *App) participantsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	names := r.FormValue("names")
	newParticipants := strings.Split(names, "\n")

	app.participantsMu.Lock()
	defer app.participantsMu.Unlock()

	app.participants = []string{}
	for _, name := range newParticipants {
		name = strings.TrimSpace(name)
		if name != "" {
			app.participants = append(app.participants, name)
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) removeParticipantHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nameToRemove := r.FormValue("name")
	if nameToRemove == "" {
		http.Error(w, "Participant name cannot be empty", http.StatusBadRequest)
		return
	}

	app.participantsMu.Lock()
	defer app.participantsMu.Unlock()

	var newParticipants []string
	for _, p := range app.participants {
		if p != nameToRemove {
			newParticipants = append(newParticipants, p)
		}
	}
	app.participants = newParticipants

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (app *App) nextParticipantHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	app.participantsMu.Lock()
	defer app.participantsMu.Unlock()

	if len(app.participants) > 0 {
		app.participants = app.participants[1:]
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) gameHandler(w http.ResponseWriter, r *http.Request) {
	participantName := strings.TrimPrefix(r.URL.Path, "/game/")

	data := struct {
		ParticipantName string
	}{
		ParticipantName: participantName,
	}

	app.templates.ExecuteTemplate(w, "game.html", data)
}

func (app *App) gameDataHandler(w http.ResponseWriter, r *http.Request) {
	participantName := strings.TrimPrefix(r.URL.Path, "/api/game-data/")

	// Try to get content from cache first
	content := app.contentCache.Pop()

	if content == nil {
		// Cache is empty - check if we should wait or generate on-demand
		if !app.contentCache.IsLoaded() {
			// Cache is still loading, wait a bit and try again
			log.Printf("Cache not loaded yet, waiting for participant %s", participantName)
			time.Sleep(2 * time.Second)
			content = app.contentCache.Pop()
		}

		if content == nil {
			// Still no content available, generate on-demand as fallback
			log.Printf("Cache empty, generating content on-demand for participant %s", participantName)

			var err error
			content, err = app.generateGameContent(r.Context())
			if err != nil {
				log.Printf("Failed to generate content on-demand: %v", err)
				http.Error(w, "Failed to generate game content", http.StatusInternalServerError)
				return
			}
		}
	}

	log.Printf("Serving game content for participant %s (cache size: %d)", participantName, app.contentCache.Size())

	data := struct {
		ParticipantName string `json:"participantName"`
		BusinessName    string `json:"businessName"`
		Slogan          string `json:"slogan"`
		Image1          string `json:"image1"`
		Image2          string `json:"image2"`
		ClappingGif     string `json:"clappingGif"`
	}{
		ParticipantName: participantName,
		BusinessName:    content.BusinessName,
		Slogan:          content.Slogan,
		Image1:          content.Image1,
		Image2:          content.Image2,
		ClappingGif:     content.ClappingGif,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (app *App) preloadCacheHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate one piece of content immediately
	content, err := app.generateGameContent(r.Context())
	if err != nil {
		log.Printf("Failed to generate content manually: %v", err)
		http.Error(w, "Failed to generate content", http.StatusInternalServerError)
		return
	}

	app.contentCache.Push(*content)
	log.Printf("Manually generated content. Cache size: %d", app.contentCache.Size())

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
