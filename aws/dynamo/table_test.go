package dynamo_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/stretchr/testify/assert"
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

	_, err = tbl.Client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("TestThings"),
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
	assert.NoError(t, err)

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
