package dynamo_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/stretchr/testify/assert"
)

func TestItem(t *testing.T) {
	dataGZ, err := dynamo.MarshalJSONGZ(map[string]any{"type": "test", "foo": "hello"})
	assert.NoError(t, err)
	assert.NotNil(t, dataGZ)

	ttl := time.Date(2030, 1, 1, 12, 0, 0, 0, time.UTC)

	// create item with all fields set
	item1 := &dynamo.Item{
		Key:    dynamo.Key{PK: "user#123", SK: "metadata"},
		OrgID:  42,
		TTL:    &ttl,
		Data:   map[string]any{"type": "test", "bar": 30},
		DataGZ: dataGZ,
		Src:    "archives",
	}

	data1, err := item1.GetData()
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"type": "test", "foo": "hello", "bar": 30}, data1)

	marshaled1, err := json.Marshal(item1)
	assert.NoError(t, err)
	assert.Equal(t, `{"PK":"user#123","SK":"metadata","OrgID":42,"TTL":"2030-01-01T12:00:00Z","Data":{"bar":30,"type":"test"},"DataGZ":"H4sIAAAAAAAA/6pWSsvPV7JSykjNyclX0lEqqSxIVbJSKkktLlGq5QIEAAD///G5VPQeAAAA","Src":"archives"}`, string(marshaled1))

	dyMap1, err := attributevalue.MarshalMap(item1)
	assert.NoError(t, err)
	assert.Equal(t, map[string]types.AttributeValue{
		"PK":    &types.AttributeValueMemberS{Value: "user#123"},
		"SK":    &types.AttributeValueMemberS{Value: "metadata"},
		"OrgID": &types.AttributeValueMemberN{Value: "42"},
		"TTL":   &types.AttributeValueMemberN{Value: "1893499200"},
		"Data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"bar":  &types.AttributeValueMemberN{Value: "30"},
			"type": &types.AttributeValueMemberS{Value: "test"},
		}},
		"DataGZ": &types.AttributeValueMemberB{Value: dataGZ},
		"Src":    &types.AttributeValueMemberS{Value: "archives"},
	}, dyMap1)

	// create item with only required fields
	item2 := &dynamo.Item{
		Key:   dynamo.Key{PK: "user#123", SK: "metadata"},
		OrgID: 42,
		Data:  map[string]any{"type": "test", "bar": 30},
	}

	data2, err := item2.GetData()
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"type": "test", "bar": 30}, data2)

	marshaled2, err := json.Marshal(item2)
	assert.NoError(t, err)
	assert.Equal(t, `{"PK":"user#123","SK":"metadata","OrgID":42,"Data":{"bar":30,"type":"test"}}`, string(marshaled2))

	dyMap2, err := attributevalue.MarshalMap(item2)
	assert.NoError(t, err)
	assert.Equal(t, map[string]types.AttributeValue{
		"PK":    &types.AttributeValueMemberS{Value: "user#123"},
		"SK":    &types.AttributeValueMemberS{Value: "metadata"},
		"OrgID": &types.AttributeValueMemberN{Value: "42"},
		"Data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"bar":  &types.AttributeValueMemberN{Value: "30"},
			"type": &types.AttributeValueMemberS{Value: "test"},
		}},
	}, dyMap2)
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
