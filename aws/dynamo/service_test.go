package dynamo_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/nyaruka/gocommon/uuids"
	"github.com/stretchr/testify/assert"
)

type Thing struct {
	uuid  uuids.UUID
	name  string
	extra map[string]any
}

type dyThing struct {
	UUID  uuids.UUID `dynamodbav:"UUID"`
	Name  string     `dynamodbav:"Name"`
	Extra []byte     `dynamodbav:"Extra"`
}

func (t *Thing) MarshalDynamo() (map[string]types.AttributeValue, error) {
	e, err := dynamo.MarshalJSONGZ(t.extra)
	if err != nil {
		return nil, fmt.Errorf("error marshaling extra: %w", err)
	}

	d := dyThing{UUID: t.uuid, Name: t.name, Extra: e}

	return attributevalue.MarshalMap(d)
}

func (t *Thing) UnmarshalDynamo(m map[string]types.AttributeValue) error {
	d := &dyThing{}

	if err := attributevalue.UnmarshalMap(m, d); err != nil {
		return fmt.Errorf("error unmarshaling thing: %w", err)
	}

	t.uuid = d.UUID
	t.name = d.Name

	if err := dynamo.UnmarshalJSONGZ(d.Extra, &t.extra); err != nil {
		return fmt.Errorf("error unmarshaling extra: %w", err)
	}

	return nil
}

func TestService(t *testing.T) {
	ctx := context.Background()

	svc, err := dynamo.NewService("root", "badkey", "us-east-1", "http://localhost:6666", "Test")
	assert.NoError(t, err)

	err = svc.Test(ctx)
	assert.ErrorContains(t, err, "exceeded maximum number of attempts, 3")

	svc, err = dynamo.NewService("root", "tembatemba", "us-east-1", "http://localhost:6000", "Test")
	assert.NoError(t, err)

	err = svc.Test(ctx)
	assert.NoError(t, err)

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

	thing1 := &Thing{uuid: "9142d9d2-bbc3-4412-b0d5-25c729c4f231", name: "One", extra: map[string]any{"foo": "bar"}}

	err = svc.PutItem(ctx, "Things", thing1)
	assert.NoError(t, err)

	thing2 := &Thing{}
	err = svc.GetItem(ctx, "Things", map[string]types.AttributeValue{"UUID": &types.AttributeValueMemberS{Value: "9142d9d2-bbc3-4412-b0d5-25c729c4f231"}}, thing2)
	assert.NoError(t, err)
	assert.Equal(t, thing1, thing2)

	_, err = svc.Client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String("TestThings")})
	assert.NoError(t, err)
}
