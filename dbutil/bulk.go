package dbutil

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// BulkQueryer is the DB/TX functionality needed for these bulk operations
type BulkQueryer interface {
	Rebind(query string) string
	QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
}

// BulkSQL takes a query which uses VALUES with struct bindings and rewrites it as a bulk operation.
// It returns the new SQL query and the args to pass to it.
func BulkSQL[T any](db BulkQueryer, sql string, structs []T) (string, []any, error) {
	if len(structs) == 0 {
		return "", nil, errors.New("can't generate bulk sql with zero structs")
	}

	// this will be our SQL placeholders for values in our final query, built dynamically
	values := strings.Builder{}
	values.Grow(7 * len(structs))

	// this will be each of the arguments to match the positional values above
	args := make([]any, 0, len(structs)*5)

	// for each value we build a bound SQL statement, then extract the values clause
	for i, value := range structs {
		valueSQL, valueArgs, err := sqlx.Named(sql, value)
		if err != nil {
			return "", nil, fmt.Errorf("error converting bulk insert args: %w", err)
		}

		args = append(args, valueArgs...)
		argValues := extractValues(valueSQL)
		if argValues == "" {
			return "", nil, fmt.Errorf("error extracting VALUES from sql: %s", valueSQL)
		}

		// append to our global values, adding comma if necessary
		values.WriteString(argValues)
		if i+1 < len(structs) {
			values.WriteString(",")
		}
	}

	valuesSQL := extractValues(sql)
	if valuesSQL == "" {
		return "", nil, fmt.Errorf("error extracting VALUES from sql: %s", sql)
	}

	return db.Rebind(strings.Replace(sql, valuesSQL, values.String(), -1)), args, nil
}

// BulkQuery runs the query as a bulk operation with the given structs
func BulkQuery[T any](ctx context.Context, db BulkQueryer, query string, structs []T) error {
	// no structs, nothing to do
	if len(structs) == 0 {
		return nil
	}

	// rewrite query as a bulk operation
	bulkQuery, args, err := BulkSQL(db, query, structs)
	if err != nil {
		return err
	}

	rows, err := db.QueryxContext(ctx, bulkQuery, args...)
	if err != nil {
		return QueryErrorWrapf(err, bulkQuery, args, "error making bulk query")
	}
	defer rows.Close()

	// if have a returning clause, read them back and try to map them
	if strings.Contains(strings.ToUpper(query), "RETURNING") {
		for i, s := range structs {
			if !rows.Next() {
				if rows.Err() != nil {
					return QueryErrorWrapf(rows.Err(), bulkQuery, args, "missing returned row for struct %d", i)
				}
				return QueryErrorf(bulkQuery, args, "missing returned row for struct %d", i)
			}

			err = rows.StructScan(s)
			if err != nil {
				return QueryErrorWrapf(err, bulkQuery, args, "error scanning returned row %d", i)
			}
		}
	}

	// iterate our remaining rows
	for rows.Next() {
	}

	// check for any error
	return QueryErrorWrapf(rows.Err(), bulkQuery, args, "error during row iteration")
}

// extractValues is just a simple utility method that extracts the portion between `VALUE(`
// and `)` in the passed in string. (leaving VALUE but not the parentheses)
func extractValues(sql string) string {
	startValues := strings.Index(sql, "VALUES(")
	if startValues <= 0 {
		return ""
	}

	// find the matching end parentheses, we need to count balances here
	openCount := 1
	endValues := -1
	for i, r := range sql[startValues+7:] {
		if r == '(' {
			openCount++
		} else if r == ')' {
			openCount--
			if openCount == 0 {
				endValues = i + startValues + 7
				break
			}
		}
	}

	if endValues <= 0 {
		return ""
	}

	return sql[startValues+6 : endValues+1]
}
