package dbutil

import (
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v9"
)

var validate = validator.New()

// ScanJSON scans a row which is JSON into a destination struct
func ScanJSON(rows *sqlx.Rows, destination interface{}) error {
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
func ScanAndValidateJSON(rows *sqlx.Rows, destination interface{}) error {
	if err := ScanJSON(rows, destination); err != nil {
		return err
	}

	err := validate.Struct(destination)
	if err != nil {
		return errors.Wrapf(err, "error validating unmarsalled JSON")
	}

	return nil
}
