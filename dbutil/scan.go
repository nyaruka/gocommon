package dbutil

import (
	"database/sql"
	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

var validate = validator.New()

// ScanJSON scans a row which is JSON into a destination struct
func ScanJSON(rows *sql.Rows, destination any) error {
	var raw json.RawMessage
	err := rows.Scan(&raw)
	if err != nil {
		return errors.Wrap(err, "error scanning row JSON")
	}

	err = json.Unmarshal(raw, destination)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling row JSON")
	}

	return nil
}

// ScanAndValidateJSON scans a row which is JSON into a destination struct and validates it
func ScanAndValidateJSON(rows *sql.Rows, destination any) error {
	if err := ScanJSON(rows, destination); err != nil {
		return err
	}

	err := validate.Struct(destination)
	if err != nil {
		return errors.Wrapf(err, "error validating unmarsalled JSON")
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
