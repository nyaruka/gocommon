package dynamo_test

import (
	"testing"

	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/stretchr/testify/assert"
)

func TestJSONGZ(t *testing.T) {
	val := map[string]any{"foo": "bar", "baz": 123.0}

	marshaled, err := dynamo.MarshalJSONGZ(val)
	assert.NoError(t, err)
	assert.NotNil(t, marshaled)

	var unmarshaled map[string]any
	err = dynamo.UnmarshalJSONGZ(marshaled, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, val, unmarshaled)
}
