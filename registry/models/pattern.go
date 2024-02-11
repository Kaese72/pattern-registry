package models

import (
	"encoding/json"
	"regexp"
)

type Pattern struct {
	ID            	int    `json:"id"`
	Pattern         string `json:"pattern"`
	Component       string `json:"component"`
	compiledPattern *regexp.Regexp
}

func (pattern *Pattern) UnmarshalJSON(bytes []byte) error {
	type Alias Pattern
	intermediary := Alias{}
	if err := json.Unmarshal(bytes, &intermediary); err != nil {
		return err
	}

	result := Pattern(intermediary)
	if err := result.Compile(); err != nil {
		return err
	}
	*pattern = result
	return nil
}

func (p *Pattern) Compile() error {
	compiledPattern, err := regexp.Compile(p.Pattern)
	if err != nil {
		return err
	}
	p.compiledPattern = compiledPattern
	return nil
}

func (p Pattern) MatchBytes(s []byte) []PatternMatch {
	match := p.compiledPattern.FindSubmatch(s)
	if len(match) == 0 {
		// No match
		return []PatternMatch{}
	}
	version := []byte{}
	for i, name := range p.compiledPattern.SubexpNames() {
		if i > 0 && i <= len(match) {
			if name == "version" {
				version = match[i]
			}
		}
	}
	return []PatternMatch{
		{
			Pattern: p,
			Version: string(version),
		},
	}
}
