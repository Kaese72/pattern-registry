package models

import (
	"context"
	"database/sql"

	"github.com/georgysavva/scany/v2/sqlscan"
)

type RegistryPattern struct {
	Pattern
	Version int `json:"version,omitempty"`
	Owner   int `json:"owner,omitempty"`
}

func DBReadRegistryPatterns(db *sql.DB) ([]RegistryPattern, error) {
	patterns := []RegistryPattern{}
	err := sqlscan.Select(context.TODO(), db, &patterns, `SELECT * FROM patterns`)
	return patterns, err
}

func DBReadRegistryPattern(db *sql.DB, id int) ([]RegistryPattern, error) {
	patterns := []RegistryPattern{}
	err := sqlscan.Select(context.TODO(), db, &patterns, `SELECT * FROM patterns WHERE id = ?`, id)
	return patterns, err
}

func DBInsertRegistryPattern(db *sql.DB, inputPattern RegistryPattern, owner int) (RegistryPattern, error) {
	resPatterns := []RegistryPattern{}
	result, err := db.Query(`INSERT INTO patterns (pattern, component, owner, version) VALUES (?, ?, ?, 1) RETURNING *`, inputPattern.Pattern.Pattern, inputPattern.Pattern.Component, owner)
	if err != nil {
		return RegistryPattern{}, err
	}
	sqlscan.ScanAll(&resPatterns, result)
	return resPatterns[0], err
}
