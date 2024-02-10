package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	registryModels "github.com/Kaese72/pattern-registry/registry/models"
	"github.com/spf13/viper"
)


type application struct {
	patterns []registryModels.Pattern
}

func (app application) runMatcher(text []byte) []registryModels.PatternMatch {
	matches := []registryModels.PatternMatch{}
	for _, pattern := range app.patterns {
		matches = append(matches, pattern.MatchBytes(text)...)
	}
	return matches
}

func (app application)handleStringParsing(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(app.runMatcher(body))
}

type ContextualizedString struct {
	Base64 string `json:"base64"`
	Context struct {
		Type string `json:"type"`
		Value string `json:"value"`
	}
}

func (app application) handleStringContextParsing(w http.ResponseWriter, r *http.Request) {
	input := ContextualizedString{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(input.Base64)
	if err != nil {
		http.Error(w, "Error decoding base64 string", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(app.runMatcher(decoded))
}

func main() {
	// Parse command line arguments and environment variables using Viper
	viper.BindEnv("PATTERN_FILE")
	viper.SetDefault("PATTERN_FILE", "patterns.json")

	// Get the value of an environment variable using Viper
	patternsFile := viper.GetString("PATTERN_FILE")

	// Read patterns file
	patternsData, err := os.ReadFile(patternsFile)
	if err != nil {
		log.Fatal(err)
	}

	// Parse patterns as Pattern struct
	var patterns []registryModels.Pattern
	err = json.Unmarshal(patternsData, &patterns)
	if err != nil {
		log.Fatal(err)
	}

	app := application{
		patterns: patterns,
	}

	http.HandleFunc("/string/match", app.handleStringParsing)
	http.HandleFunc("/string/context/match", app.handleStringContextParsing)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

