package elastic_test

import (
	"testing"

	"github.com/nyaruka/gocommon/elastic"
	"github.com/nyaruka/gocommon/jsonx"
	"github.com/stretchr/testify/assert"
)

func TestQuery(t *testing.T) {
	tcs := []struct {
		q    elastic.Query
		json []byte
	}{
		{elastic.Ids("235", "465", "787"), []byte(`{"ids": {"values": ["235", "465", "787"]}}`)},
		{elastic.Term("age", 42), []byte(`{"term": {"age": {"value":42}}}`)},
		{elastic.Exists("age"), []byte(`{"exists": {"field": "age"}}`)},
		{elastic.Match("name", "Bob"), []byte(`{"match": {"name": {"query": "Bob"}}}`)},
		{elastic.MatchPhrase("name", "Bob"), []byte(`{"match_phrase": {"name": {"query": "Bob"}}}`)},
		{elastic.GreaterThan("age", 45), []byte(`{"range": {"age": {"gt": 45}}}`)},
		{elastic.GreaterThanOrEqual("age", 45), []byte(`{"range": {"age": {"gte": 45}}}`)},
		{elastic.LessThan("age", 45), []byte(`{"range": {"age": {"lt": 45}}}`)},
		{elastic.LessThanOrEqual("age", 45), []byte(`{"range": {"age": {"lte": 45}}}`)},
		{elastic.Between("age", 20, 45), []byte(`{"range": {"age": {"gte": 20, "lt": 45}}}`)},
		{
			elastic.Any(elastic.Ids("235"), elastic.Term("age", 42)),
			[]byte(`{"bool": {"should": [{"ids": {"values": ["235"]}}, {"term": {"age": {"value":42}}}]}}`),
		},
		{
			elastic.All(elastic.Ids("235"), elastic.Term("age", 42)),
			[]byte(`{"bool": {"must": [{"ids": {"values": ["235"]}}, {"term": {"age": {"value":42}}}]}}`),
		},
		{
			elastic.Not(elastic.Ids("235")),
			[]byte(`{"bool": {"must_not": {"ids": {"values": ["235"]}}}}`),
		},
		{
			elastic.Bool([]elastic.Query{elastic.Ids("235"), elastic.Term("age", 42)}, []elastic.Query{elastic.Exists("age")}),
			[]byte(`{"bool": {"must": [{"ids": {"values": ["235"]}}, {"term": {"age": {"value":42}}}], "must_not": [{"exists": {"field": "age"}}]}}`),
		},
		{
			elastic.Bool([]elastic.Query{}, []elastic.Query{elastic.Exists("age")}),
			[]byte(`{"bool": {"must_not": [{"exists": {"field": "age"}}]}}`),
		},
		{
			elastic.Bool([]elastic.Query{elastic.Ids("235"), elastic.Term("age", 42)}, []elastic.Query{}),
			[]byte(`{"bool": {"must": [{"ids": {"values": ["235"]}}, {"term": {"age": {"value":42}}}]}}`),
		},
		{elastic.Nested("group", elastic.Term("group.id", 10)), []byte(`{"nested": {"path": "group", "query": {"term": {"group.id": {"value":10}}}}}`)},
	}

	for i, tc := range tcs {
		assert.JSONEq(t, string(tc.json), string(jsonx.MustMarshal(tc.q)), "%d: elastic mismatch", i)
	}
}
