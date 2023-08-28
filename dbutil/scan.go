package dbutil

import "database/sql"

// SliceScan scans a single value from each row into the given slice
func SliceScan[V any](rows *sql.Rows, s []V) ([]V, error) {
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

// MapScan scans a key and value from each row into the given map
func MapScan[K comparable, V any](rows *sql.Rows, m map[K]V) error {
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
