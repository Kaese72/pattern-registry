package client

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Kaese72/pattern-registry/registry/models"
)

type Client struct {
	// BaseUrl is assumed to have a / at the end of it
	BaseUrl string
}

func (client Client) GetPatterns() ([]models.RegistryPattern, error) {
	response, err := http.Get(client.BaseUrl + "patterns")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("Error from server: " + response.Status)
	}
	patterns := []models.RegistryPattern{}
	if err := json.NewDecoder(response.Body).Decode(&patterns); err != nil {
		return nil, err
	}
	return patterns, nil
}
