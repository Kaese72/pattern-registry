package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Kaese72/pattern-registry/registry/models"
	"github.com/georgysavva/scany/v2/sqlscan"
)

func DBReadRegistryPatterns(db *sql.DB) ([]models.RegistryPattern, error) {
	patterns := []models.RegistryPattern{}
	err := sqlscan.Select(context.TODO(), db, &patterns, `SELECT * FROM patterns`)
	return patterns, err
}

func DBReadRegistryPattern(db *sql.DB, id int) (models.RegistryPattern, error) {
	patterns := []models.RegistryPattern{}
	err := sqlscan.Select(context.TODO(), db, &patterns, `SELECT * FROM patterns WHERE id = ?`, id)
	if err != nil {
		return models.RegistryPattern{}, err
	}
	if len(patterns) == 0 {
		return models.RegistryPattern{}, errors.New("no pattern found")
	}
	return patterns[0], nil
}

func DBInsertRegistryPattern(db *sql.DB, inputPattern models.RegistryPattern, owner int) (models.RegistryPattern, error) {
	resPatterns := []models.RegistryPattern{}
	result, err := db.Query(`INSERT INTO patterns (pattern, component, owner, version) VALUES (?, ?, ?, 1) RETURNING *`, inputPattern.Pattern.Pattern, inputPattern.Pattern.Component, owner)
	if err != nil {
		return models.RegistryPattern{}, err
	}
	err = sqlscan.ScanAll(&resPatterns, result)
	if err != nil {
		return models.RegistryPattern{}, err
	}
	if len(resPatterns) == 0 {
		return models.RegistryPattern{}, errors.New("no pattern returned from insert")
	}
	return resPatterns[0], nil
}

func DBUpdateRegistryPattern(db *sql.DB, inputPattern models.Pattern, owner int, id int) (models.RegistryPattern, error) {
	_, err := db.Exec(`UPDATE patterns SET pattern = ? WHERE id = ?`, inputPattern.Pattern, id)
	if err != nil {
		return models.RegistryPattern{}, err
	}
	return DBReadRegistryPattern(db, id)
}

func DBDeleteRegistryPattern(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM patterns WHERE id = ?`, id)
	return err
}
