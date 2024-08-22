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

func TestService(t *testing.T) {
	ctx := context.Background()

	svc, err := dynamo.NewService("root", "tembatemba", "us-east-1", "http://localhost:6000", "Test")
	assert.NoError(t, err)

	err = svc.Test(ctx, "Things")
	assert.ErrorContains(t, err, "ResourceNotFoundException")

	_, err = svc.Client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("TestThings"),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("UUID"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("UUID"), AttributeType: types.ScalarAttributeTypeS},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	assert.NoError(t, err)

	err = svc.Test(ctx, "Things")
	assert.NoError(t, err)

	_, err = svc.Client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String("TestThings")})
	assert.NoError(t, err)
}
