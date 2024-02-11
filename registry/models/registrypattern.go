package models

type RegistryPattern struct {
	Pattern
	Version int `json:"version,omitempty"`
	Owner   int `json:"owner,omitempty"`
}
