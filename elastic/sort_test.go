package elastic_test

import (
	"testing"

	"github.com/nyaruka/gocommon/elastic"
	"github.com/nyaruka/gocommon/jsonx"
	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	tcs := []struct {
		q    elastic.Sort
		json []byte
	}{
		{elastic.SortBy("name", true), []byte(`{"name": {"order": "asc"}}`)},
		{elastic.SortBy("name", false), []byte(`{"name": {"order": "desc"}}`)},
		{
			elastic.SortNested("age", elastic.Term("fields.field", "1234"), "fields", true),
			[]byte(`{"age": {"nested": {"filter": {"term": {"fields.field": "1234"}}, "path": "fields"}, "order":"asc"}}`),
		},
	}

	for i, tc := range tcs {
		assert.JSONEq(t, string(tc.json), string(jsonx.MustMarshal(tc.q)), "%d: elastic mismatch", i)
	}
}
