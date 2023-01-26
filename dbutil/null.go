package dbutil

import (
	"database/sql/driver"

	"github.com/pkg/errors"
)

// ScanNullString scans a nullable CHAR/TEXT into a string type using empty string for NULL
func ScanNullString[T ~string](v any, s *T) error {
	if v == nil {
		*s = ""
		return nil
	}
	t, ok := v.(string)
	if ok {
		*s = T(t)
		return nil
	}

	return errors.Errorf("unable to scan %T as %T", v, s)
}

// NullStringValue converts a string type value to NULL if it is empty
func NullStringValue[T ~string](s T) (driver.Value, error) {
	if s == "" {
		return nil, nil
	}
	return string(s), nil
}
