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

	tbl, err := dynamo.NewTable[ThingKey, ThingItem]("root", "badkey", "us-east-1", "http://localhost:6666", "TestThings")
	assert.NoError(t, err)

	err = tbl.Test(ctx)
	assert.ErrorContains(t, err, "exceeded maximum number of attempts, 3")

	tbl, err = dynamo.NewTable[ThingKey, ThingItem]("root", "tembatemba", "us-east-1", "http://localhost:6000", "TestThings")
	assert.NoError(t, err)
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

	thing1 := &ThingItem{ThingKey: ThingKey{PK: "11", SK: "22"}, Name: "Test Thing", Count: 42}

	err = tbl.PutItem(ctx, thing1)
	assert.NoError(t, err)

	thing2, err := tbl.GetItem(ctx, ThingKey{PK: "11", SK: "22"})
	assert.NoError(t, err)
	assert.Equal(t, thing1, thing2)

	err = tbl.Delete(ctx)
	assert.NoError(t, err)
}
