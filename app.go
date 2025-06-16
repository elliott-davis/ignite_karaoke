package main

import (
	"html/template"
	"sync"
	"time"
)

// App holds the application dependencies and state
type App struct {
	participants     []string
	participantsMu   sync.Mutex
	templates        *template.Template
	usedGifs         map[string]bool
	usedGifsMu       sync.Mutex
	giphyAPIKey      string
	giphyCache       []string
	giphyCacheMu     sync.Mutex
	giphyCacheExpiry time.Time
	googleAPIKey     string
	generator        Generator
	// We can add clients for external services here later
}

var _ GiphyClient = (*App)(nil)
