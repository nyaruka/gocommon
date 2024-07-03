package dbutil

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Scanable is an interface to allow scanning of sql.Row or sql.Rows
type Scannable interface {
	Scan(dest ...any) error
}

// ScanJSON scans a row which is JSON into a destination struct
func ScanJSON(src Scannable, dest any) error {
	var raw json.RawMessage

	if err := src.Scan(&raw); err != nil {
		return fmt.Errorf("error scanning row JSON: %w", err)
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("error unmarshalling row JSON: %w", err)
	}

	return nil
}

// ScanAndValidateJSON scans a row which is JSON into a destination struct and validates it
func ScanAndValidateJSON(src Scannable, dest any) error {
	if err := ScanJSON(src, dest); err != nil {
		return err
	}

	err := validate.Struct(dest)
	if err != nil {
		return fmt.Errorf("error validating unmarsalled JSON: %w", err)
	}

	return nil
}

// ScanAllJSON scans all rows as a single column containing JSON that be unmarshalled into instances of V.
func ScanAllJSON[V any](rows *sql.Rows, s []V) ([]V, error) {
	defer rows.Close()

	var v V

	for rows.Next() {
		if err := ScanJSON(rows, &v); err != nil {
			return nil, err
		}
		s = append(s, v)
	}
	return s, rows.Err()
}

// ScanAllSlice scans all rows as a single value and returns them in the given slice.
func ScanAllSlice[V any](rows *sql.Rows, s []V) ([]V, error) {
	defer rows.Close()

	var v V

	for rows.Next() {
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		s = append(s, v)
	}
	return s, rows.Err()
}

// ScanAllMap scans a key and value from each two column row into the given map
func ScanAllMap[K comparable, V any](rows *sql.Rows, m map[K]V) error {
	defer rows.Close()

	var k K
	var v V

	for rows.Next() {
		if err := rows.Scan(&k, &v); err != nil {
			return err
		}
		m[k] = v
	}
	return rows.Err()
}
