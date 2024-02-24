package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/Kaese72/pattern-registry/apierrors"
	"github.com/Kaese72/pattern-registry/registry/models"
	"github.com/georgysavva/scany/v2/sqlscan"
)

var registryPatternFilterConverters = map[string]func(Filter) (string, error){}

func init() {
	registryPatternFilterConverters["id"] = func(filter Filter) (string, error) { return filter.Number() }
	registryPatternFilterConverters["pattern"] = func(filter Filter) (string, error) { return filter.String() }
	registryPatternFilterConverters["component"] = func(filter Filter) (string, error) { return filter.String() }
	registryPatternFilterConverters["owner"] = func(filter Filter) (string, error) { return filter.Number() }
	registryPatternFilterConverters["version"] = func(filter Filter) (string, error) { return filter.Number() }
}

func DBRegistryPatternFilter(filters []Filter) (string, []interface{}, error) {
	queryFragments := []string{}
	args := []interface{}{}
	for _, filter := range filters {
		converter, ok := registryPatternFilterConverters[filter.Key]
		if !ok {
			return "", nil, apierrors.APIError{Code: 400, WrappedError: errors.New("attribute may not be filtered on")}
		}
		converted, err := converter(filter)
		if err != nil {
			return "", nil, apierrors.APIError{Code: 400, WrappedError: err}
		}
		queryFragments = append(queryFragments, converted)
		args = append(args, filter.Value)
	}
	return strings.Join(queryFragments, " AND "), args, nil
}

func DBReadRegistryPatterns(db *sql.DB, filters []Filter) ([]models.RegistryPattern, error) {
	patterns := []models.RegistryPattern{}
	query := `SELECT * FROM patterns`
	variables := []interface{}{}
	if queryQuery, queryVariables, err := DBRegistryPatternFilter(filters); err == nil {
		if queryQuery != "" {
			query += " WHERE " + queryQuery
			variables = queryVariables
		}
	} else {
		return nil, err
	}
	err := sqlscan.Select(context.TODO(), db, &patterns, query, variables...)
	return patterns, err
}

func DBReadRegistryPattern(db *sql.DB, id int) (models.RegistryPattern, error) {
	patterns := []models.RegistryPattern{}
	err := sqlscan.Select(context.TODO(), db, &patterns, `SELECT * FROM patterns WHERE id = ?`, id)
	if err != nil {
		return models.RegistryPattern{}, err
	}
	if len(patterns) == 0 {
		return models.RegistryPattern{}, apierrors.APIError{Code: 404, WrappedError: errors.New("pattern not found")}
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
