package dynamo_test

import (
	"context"
	"testing"
	"time"

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

	// Wait a bit for table to be ready
	time.Sleep(50 * time.Millisecond)
}

func TestTable(t *testing.T) {
	ctx := context.Background()

	client, err := dynamo.NewClient("root", "badkey", "us-east-1", "http://localhost:6666")
	assert.NoError(t, err)

	tbl := dynamo.NewTable[ThingKey, ThingItem](client, "TestThings")
	err = tbl.Test(ctx)
	assert.ErrorContains(t, err, "exceeded maximum number of attempts, 3")

	client, err = dynamo.NewClient("root", "tembatemba", "us-east-1", "http://localhost:6000")
	assert.NoError(t, err)

	tbl = dynamo.NewTable[ThingKey, ThingItem](client, "TestThings")
	assert.Equal(t, "TestThings", tbl.Name())

	err = tbl.Test(ctx)
	assert.ErrorContains(t, err, "Cannot do operations on a non-existent table")

	createTestTable(t, client, "TestThings")

	err = tbl.Test(ctx)
	assert.NoError(t, err)

	thing1 := &ThingItem{ThingKey: ThingKey{PK: "P11", SK: "SAA"}, Name: "Test Thing 1", Count: 42}
	thing2 := &ThingItem{ThingKey: ThingKey{PK: "P22", SK: "SBB"}, Name: "Test Thing 2", Count: 235}

	err = tbl.PutItem(ctx, thing1)
	assert.NoError(t, err)
	err = tbl.PutItem(ctx, thing2)
	assert.NoError(t, err)

	count, err := tbl.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	read, err := tbl.GetItem(ctx, ThingKey{PK: "P11", SK: "SAA"})
	assert.NoError(t, err)
	assert.Equal(t, thing1, read)

	err = tbl.Purge(ctx)
	assert.NoError(t, err)

	count, err = tbl.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	err = tbl.Delete(ctx)
	assert.NoError(t, err)

	_, err = tbl.Count(ctx)
	assert.ErrorContains(t, err, "non-existent table")
}

func TestBatchWriteItem(t *testing.T) {
	ctx := context.Background()

	client, err := dynamo.NewClient("root", "tembatemba", "us-east-1", "http://localhost:6000")
	assert.NoError(t, err)

	createTestTable(t, client, "TestBatchThings")
	tbl := dynamo.NewTable[ThingKey, ThingItem](client, "TestBatchThings")

	defer tbl.Delete(ctx)

	// Test with empty slice
	unprocessed, err := tbl.BatchWriteItem(ctx, []*ThingItem{})
	assert.NoError(t, err)
	assert.Nil(t, unprocessed)

	// Test with multiple items
	items := []*ThingItem{
		{ThingKey: ThingKey{PK: "BATCH1", SK: "S1"}, Name: "Batch Item 1", Count: 10},
		{ThingKey: ThingKey{PK: "BATCH2", SK: "S2"}, Name: "Batch Item 2", Count: 20},
		{ThingKey: ThingKey{PK: "BATCH3", SK: "S3"}, Name: "Batch Item 3", Count: 30},
		{ThingKey: ThingKey{PK: "BATCH4", SK: "S4"}, Name: "Batch Item 4", Count: 40},
	}

	unprocessed, err = tbl.BatchWriteItem(ctx, items)
	assert.NoError(t, err)
	assert.Empty(t, unprocessed) // All items should be processed successfully

	// Verify items were written
	count, err := tbl.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 4, count)

	// Verify individual items can be retrieved
	for _, item := range items {
		retrieved, err := tbl.GetItem(ctx, item.ThingKey)
		assert.NoError(t, err)
		assert.Equal(t, item, retrieved)
	}

	// Test with single item
	singleItem := []*ThingItem{
		{ThingKey: ThingKey{PK: "SINGLE", SK: "S1"}, Name: "Single Item", Count: 100},
	}

	unprocessed, err = tbl.BatchWriteItem(ctx, singleItem)
	assert.NoError(t, err)
	assert.Empty(t, unprocessed)

	count, err = tbl.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}
