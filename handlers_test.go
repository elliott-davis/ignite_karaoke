package main

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// MockGenerator is a mock implementation of the Generator interface for testing.
type MockGenerator struct{}

func (m *MockGenerator) GenerateBusinessIdea(ctx context.Context) (string, string, error) {
	return "Test Business", "Test Slogan", nil
}

func (m *MockGenerator) GenerateImagePrompt(ctx context.Context) (string, error) {
	return "a test image prompt", nil
}

func (m *MockGenerator) GenerateImage(ctx context.Context, prompt string) (string, error) {
	return "data:image/png;base64,test", nil
}

func TestParticipantsHandler(t *testing.T) {
	app := &App{
		templates: template.Must(template.ParseFS(templateFS, "templates/*.html")),
	}

	form := url.Values{}
	form.Add("names", "Alice\nBob\nCharlie")

	req, err := http.NewRequest("POST", "/participants", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.participantsHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusSeeOther)
	}

	if len(app.participants) != 3 {
		t.Errorf("expected 3 participants, got %d", len(app.participants))
	}

	expectedParticipants := []string{"Alice", "Bob", "Charlie"}
	for i, p := range app.participants {
		if p != expectedParticipants[i] {
			t.Errorf("expected participant %s, got %s", expectedParticipants[i], p)
		}
	}
}

func TestGameDataHandler(t *testing.T) {
	app := &App{
		generator: &MockGenerator{},
	}

	req, err := http.NewRequest("GET", "/api/game-data/test-participant", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.gameDataHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Add more assertions here to check the response body
}
