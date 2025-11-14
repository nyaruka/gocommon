package dynamo_test

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/nyaruka/gocommon/aws/dynamo/dyntest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ThingKey struct {
	PK string `dynamodbav:"PK"`
	SK string `dynamodbav:"SK"`
}

type ThingItem struct {
	ThingKey

	Name  string `dynamodbav:"Name"`
	Count int    `dynamodbav:"Count"`
}

func createTestTable(t *testing.T, client *dynamodb.Client, name string) {
	_, err := client.CreateTable(t.Context(), &dynamodb.CreateTableInput{
		TableName: aws.String(name),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	require.NoError(t, err)
}

func TestPutAndGet(t *testing.T) {
	ctx := t.Context()

	client, err := dynamo.NewClient("root", "tembatemba", "us-east-1", "http://localhost:6000")
	assert.NoError(t, err)

	defer dyntest.Drop(t, client, "TestThings")

	err = dynamo.Test(ctx, client, "TestThings")
	assert.Error(t, err)

	createTestTable(t, client, "TestThings")

	err = dynamo.Test(ctx, client, "TestThings")
	assert.NoError(t, err)

	thing1 := &dynamo.Item{Key: dynamo.Key{PK: "P11", SK: "SAA"}, OrgID: 1, Data: map[string]any{"name": "Thing 1"}}
	thing2 := &dynamo.Item{Key: dynamo.Key{PK: "P22", SK: "SBB"}, OrgID: 1, Data: map[string]any{"name": "Thing 2"}}

	err = dynamo.PutItem(ctx, client, "TestThings", thing1)
	assert.NoError(t, err)
	err = dynamo.PutItem(ctx, client, "TestThings", thing2)
	assert.NoError(t, err)

	dyntest.AssertCount(t, client, "TestThings", 2)

	obj, err := dynamo.GetItem(ctx, client, "TestThings", dynamo.Key{PK: "P11", SK: "SAA"})
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"name": "Thing 1"}, obj.Data)

	// try to get a non-existent item
	obj, err = dynamo.GetItem(ctx, client, "TestThings", dynamo.Key{PK: "P77", SK: "SAA"})
	assert.NoError(t, err)
	assert.Nil(t, obj)

	unprocessed, err := dynamo.BatchPutItem(ctx, client, "TestThings", []*dynamo.Item{})
	assert.NoError(t, err)
	assert.Equal(t, []*dynamo.Item{}, unprocessed)

	unprocessed, err = dynamo.BatchPutItem(ctx, client, "TestThings", []*dynamo.Item{
		{Key: dynamo.Key{PK: "BATCH1", SK: "S1"}, OrgID: 1, Data: map[string]any{"name": "Batch Item 1"}},
		{Key: dynamo.Key{PK: "BATCH2", SK: "S2"}, OrgID: 1, Data: map[string]any{"name": "Batch Item 2"}},
		{Key: dynamo.Key{PK: "BATCH3", SK: "S3"}, OrgID: 1, Data: map[string]any{"name": "Batch Item 3"}},
		{Key: dynamo.Key{PK: "BATCH4", SK: "S4"}, OrgID: 1, Data: map[string]any{"name": "Batch Item 4"}},
	})
	assert.NoError(t, err)
	assert.Empty(t, unprocessed)

	dyntest.AssertCount(t, client, "TestThings", 6)
}
