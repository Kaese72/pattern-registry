package models

type PatternMatch struct {
	Pattern Pattern `json:"pattern"`
	Version string  `json:"version,omitempty"`
}