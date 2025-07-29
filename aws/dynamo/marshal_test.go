package dynamo_test

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/stretchr/testify/assert"
)

type byMethods struct {
	N string
}

func (o *byMethods) MarshalDynamo() (map[string]types.AttributeValue, error) {
	return map[string]types.AttributeValue{
		"Name": &types.AttributeValueMemberS{Value: o.N},
	}, nil
}

func (o *byMethods) UnmarshalDynamo(d map[string]types.AttributeValue) error {
	o.N = d["Name"].(*types.AttributeValueMemberS).Value
	return nil
}

type byAttributes struct {
	N string `dynamodbav:"Name"`
}

func TestMarshaling(t *testing.T) {
	o1 := &byMethods{N: "By Methods"}
	m1, err := dynamo.Marshal(o1)
	assert.NoError(t, err)
	assert.Equal(t, map[string]types.AttributeValue{
		"Name": &types.AttributeValueMemberS{Value: "By Methods"},
	}, m1)

	var u1 byMethods
	err = dynamo.Unmarshal(m1, &u1)
	assert.NoError(t, err)
	assert.Equal(t, "By Methods", u1.N)

	o2 := &byAttributes{N: "By Attributes"}
	m2, err := dynamo.Marshal(o2)
	assert.NoError(t, err)
	assert.Equal(t, map[string]types.AttributeValue{
		"Name": &types.AttributeValueMemberS{Value: "By Attributes"},
	}, m2)

	var u2 byAttributes
	err = dynamo.Unmarshal(m2, &u2)
	assert.NoError(t, err)
	assert.Equal(t, "By Attributes", u2.N)
}
