package dynamo_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/aws/dynamo"
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
	_, err := client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
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
	ctx := context.Background()

	client, err := dynamo.NewClient("root", "tembatemba", "us-east-1", "http://localhost:6000")
	assert.NoError(t, err)

	defer dynamo.Drop(ctx, client, "TestThings")

	err = dynamo.Test(ctx, client, "TestThings")
	assert.Error(t, err)

	createTestTable(t, client, "TestThings")

	err = dynamo.Test(ctx, client, "TestThings")
	assert.NoError(t, err)

	thing1 := &ThingItem{ThingKey: ThingKey{PK: "P11", SK: "SAA"}, Name: "Test Thing 1", Count: 42}
	thing2 := &ThingItem{ThingKey: ThingKey{PK: "P22", SK: "SBB"}, Name: "Test Thing 2", Count: 235}

	err = dynamo.PutItem(ctx, client, "TestThings", thing1)
	assert.NoError(t, err)
	err = dynamo.PutItem(ctx, client, "TestThings", thing2)
	assert.NoError(t, err)

	count, err := dynamo.Count(ctx, client, "TestThings")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	obj, err := dynamo.GetItem[ThingKey, ThingItem](ctx, client, "TestThings", ThingKey{PK: "P11", SK: "SAA"})
	assert.NoError(t, err)
	assert.NotNil(t, obj)

	unprocessed, err := dynamo.BatchPutItem(ctx, client, "TestThings", []*ThingItem{})
	assert.NoError(t, err)
	assert.Nil(t, unprocessed)

	unprocessed, err = dynamo.BatchPutItem(ctx, client, "TestThings", []*ThingItem{
		{ThingKey: ThingKey{PK: "BATCH1", SK: "S1"}, Name: "Batch Item 1", Count: 10},
		{ThingKey: ThingKey{PK: "BATCH2", SK: "S2"}, Name: "Batch Item 2", Count: 20},
		{ThingKey: ThingKey{PK: "BATCH3", SK: "S3"}, Name: "Batch Item 3", Count: 30},
		{ThingKey: ThingKey{PK: "BATCH4", SK: "S4"}, Name: "Batch Item 4", Count: 40},
	})
	assert.NoError(t, err)
	assert.Empty(t, unprocessed)

	count, err = dynamo.Count(ctx, client, "TestThings")
	assert.NoError(t, err)
	assert.Equal(t, 6, count)

	err = dynamo.Truncate(ctx, client, "TestThings")
	assert.NoError(t, err)

	count, err = dynamo.Count(ctx, client, "TestThings")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	err = dynamo.Drop(ctx, client, "TestThings")
	assert.NoError(t, err)

	_, err = dynamo.Count(ctx, client, "TestThings")
	assert.ErrorContains(t, err, "non-existent table")
}
