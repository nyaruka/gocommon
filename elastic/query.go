package elastic

type Query map[string]any

// Not is a shortcut for an ids query
func Ids(values ...string) Query {
	return Query{"ids": map[string]any{"values": values}}
}

// Term is a shortcut for a term query
func Term(field string, value any) Query {
	return Query{"term": map[string]any{field: map[string]any{"value": value}}}
}

// Exists is a shortcut for an exists query
func Exists(field string) Query {
	return Query{"exists": map[string]any{"field": field}}
}

// Match is a shortcut for a match query
func Match(field string, value any) Query {
	return Query{"match": map[string]any{field: map[string]any{"query": value}}}
}

// MatchAll is a shortcut for a match all query
func MatchAll() Query {
	return Query{"match_all": map[string]any{}}
}

// MatchPhrase is a shortcut for a match_phrase query
func MatchPhrase(field, value string) Query {
	return Query{"match_phrase": map[string]any{field: map[string]any{"query": value}}}
}

// GreaterThan is a shortcut for a range query where x > value
func GreaterThan(field string, value any) Query {
	return Query{
		"range": map[string]any{
			field: map[string]any{
				"gt": value,
			},
		},
	}
}

// GreaterThanOrEqual is a shortcut for a range query where x >= value
func GreaterThanOrEqual(field string, value any) Query {
	return Query{
		"range": map[string]any{
			field: map[string]any{
				"gte": value,
			},
		},
	}
}

// LessThan is a shortcut for a range query where x < value
func LessThan(field string, value any) Query {
	return Query{
		"range": map[string]any{
			field: map[string]any{
				"lt": value,
			},
		},
	}
}

// LessThanOrEqual is a shortcut for a range query where x <= value
func LessThanOrEqual(field string, value any) Query {
	return Query{
		"range": map[string]any{
			field: map[string]any{
				"lte": value,
			},
		},
	}
}

// Between is a shortcut for a range query where from <= x < to
func Between(field string, from, to any) Query {
	return Query{
		"range": map[string]any{
			field: map[string]any{
				"gte": from,
				"lt":  to,
			},
		},
	}
}

// Any is a shortcut for a bool query with a should clause
func Any(queries ...Query) Query {
	return Query{"bool": map[string]any{"should": queries}}
}

// All is a shortcut for a bool query with a must clause
func All(queries ...Query) Query {
	return Query{"bool": map[string]any{"must": queries}}
}

// Not is a shortcut for a bool query with a must_not clause
func Not(query Query) Query {
	return Query{"bool": map[string]any{"must_not": query}}
}

// Bool is a shortcut for a bool query with multiple must and must_not clauses
func Bool(all []Query, none []Query) Query {
	bq := map[string]any{}

	if len(all) > 0 {
		bq["must"] = all
	}
	if len(none) > 0 {
		bq["must_not"] = none
	}

	return Query{"bool": bq}
}

// Nested is a shortcut for a nested query
func Nested(path string, query Query) Query {
	return Query{"nested": map[string]any{"path": path, "query": query}}
}
