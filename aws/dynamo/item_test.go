package dynamo_test

import (
	"encoding/json"
	"testing"

	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/stretchr/testify/assert"
)

func TestItem(t *testing.T) {
	dataGZ, err := dynamo.MarshalJSONGZ(map[string]any{"type": "test", "foo": "hello"})
	assert.NoError(t, err)
	assert.NotNil(t, dataGZ)

	item1 := &dynamo.Item{
		Key:    dynamo.Key{PK: "user#123", SK: "metadata"},
		OrgID:  42,
		Data:   map[string]any{"type": "test", "bar": 30},
		DataGZ: dataGZ,
	}

	data, err := item1.GetData()
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"type": "test", "foo": "hello", "bar": 30}, data)

	marshaled, err := json.Marshal(item1)
	assert.NoError(t, err)
	assert.Equal(t, `{"PK":"user#123","SK":"metadata","OrgID":42,"Data":{"bar":30,"type":"test"},"DataGZ":"H4sIAAAAAAAA/6pWSsvPV7JSykjNyclX0lEqqSxIVbJSKkktLlGq5QIEAAD///G5VPQeAAAA"}`, string(marshaled))
}

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
