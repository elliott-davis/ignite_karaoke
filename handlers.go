package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
		Participants []string
	}{
		Participants: app.participants,
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

	businessName, slogan, err := app.generator.GenerateBusinessIdea(r.Context())
	if err != nil {
		http.Error(w, "Failed to generate business idea", http.StatusInternalServerError)
		return
	}

	clappingGif, err := app.GetClappingGiphy(r.Context())
	if err != nil {
		fmt.Printf("Failed to get clapping gif: %s\n", err)
		clappingGif = "https://media.giphy.com/media/3o7abB06u9bNzA8lu8/giphy.gif"
	}

	imagePrompt1, err := app.generator.GenerateImagePrompt(r.Context())
	if err != nil {
		log.Printf("Failed to generate image prompt 1: %v", err)
		http.Error(w, "Failed to generate image prompt 1", http.StatusInternalServerError)
		return
	}
	log.Printf("Generated Image Prompt 1: %s", imagePrompt1)
	image1, err := app.generator.GenerateImage(r.Context(), imagePrompt1)
	if err != nil {
		log.Printf("Failed to generate image 1 with prompt '%s': %v", imagePrompt1, err)
		http.Error(w, "Failed to generate image 1", http.StatusInternalServerError)
		return
	}

	imagePrompt2, err := app.generator.GenerateImagePrompt(r.Context())
	if err != nil {
		log.Printf("Failed to generate image prompt 2: %v", err)
		http.Error(w, "Failed to generate image prompt 2", http.StatusInternalServerError)
		return
	}
	log.Printf("Generated Image Prompt 2: %s", imagePrompt2)
	image2, err := app.generator.GenerateImage(r.Context(), imagePrompt2)
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
