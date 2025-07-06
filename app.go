package main

import (
	"html/template"
	"sync"
	"time"
)

// GameContent represents a complete set of content for one game session
type GameContent struct {
	BusinessName string
	Slogan       string
	Image1       string
	Image2       string
	ClappingGif  string
	CreatedAt    time.Time
}

// ContentCache holds pre-generated game content
type ContentCache struct {
	items    []GameContent
	mu       sync.RWMutex
	maxSize  int
	isLoaded bool
}

// NewContentCache creates a new content cache with specified max size
func NewContentCache(maxSize int) *ContentCache {
	return &ContentCache{
		items:   make([]GameContent, 0, maxSize),
		maxSize: maxSize,
	}
}

// Pop removes and returns the first item from cache, or nil if empty
func (cc *ContentCache) Pop() *GameContent {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if len(cc.items) == 0 {
		return nil
	}

	item := cc.items[0]
	cc.items = cc.items[1:]
	return &item
}

// Push adds an item to the end of the cache, removing oldest if at capacity
func (cc *ContentCache) Push(content GameContent) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if len(cc.items) >= cc.maxSize {
		cc.items = cc.items[1:] // Remove oldest
	}

	cc.items = append(cc.items, content)
}

// Size returns the current number of cached items
func (cc *ContentCache) Size() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return len(cc.items)
}

// SetLoaded marks the cache as having been initially loaded
func (cc *ContentCache) SetLoaded() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.isLoaded = true
}

// IsLoaded returns whether the cache has been initially loaded
func (cc *ContentCache) IsLoaded() bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.isLoaded
}

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
	contentCache     *ContentCache
	preloadStop      chan struct{}
	preloadRunning   bool
	preloadMu        sync.Mutex
	// We can add clients for external services here later
}

var _ GiphyClient = (*App)(nil)
