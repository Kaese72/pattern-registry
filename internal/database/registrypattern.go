package database

import (
	"context"
	"database/sql"

	"github.com/Kaese72/pattern-registry/registry/models"
	"github.com/georgysavva/scany/v2/sqlscan"
)

func DBReadRegistryPatterns(db *sql.DB) ([]models.RegistryPattern, error) {
	patterns := []models.RegistryPattern{}
	err := sqlscan.Select(context.TODO(), db, &patterns, `SELECT * FROM patterns`)
	return patterns, err
}

func DBReadRegistryPattern(db *sql.DB, id int) ([]models.RegistryPattern, error) {
	patterns := []models.RegistryPattern{}
	err := sqlscan.Select(context.TODO(), db, &patterns, `SELECT * FROM patterns WHERE id = ?`, id)
	return patterns, err
}

func DBInsertRegistryPattern(db *sql.DB, inputPattern models.RegistryPattern, owner int) (models.RegistryPattern, error) {
	resPatterns := []models.RegistryPattern{}
	result, err := db.Query(`INSERT INTO patterns (pattern, component, owner, version) VALUES (?, ?, ?, 1) RETURNING *`, inputPattern.Pattern.Pattern, inputPattern.Pattern.Component, owner)
	if err != nil {
		return models.RegistryPattern{}, err
	}
	sqlscan.ScanAll(&resPatterns, result)
	return resPatterns[0], err
}