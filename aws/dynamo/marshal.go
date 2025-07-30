package dynamo

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Marshaler lets types declare their own DynamoDB marshaling logic.
type Marshaler interface {
	MarshalDynamo() (map[string]types.AttributeValue, error)
}

// Unmarshaler lets types declare their own DynamoDB unmarshaling logic.
type Unmarshaler interface {
	UnmarshalDynamo(map[string]types.AttributeValue) error
}

// Marshal marshals a value into a DynamoDB attribute map.
func Marshal(item any) (map[string]types.AttributeValue, error) {
	marshaler, ok := item.(Marshaler)
	if ok {
		return marshaler.MarshalDynamo()
	}
	return attributevalue.MarshalMap(item)
}

// Unmarshal unmarshals a value from a DynamoDB attribute map.
func Unmarshal(attrs map[string]types.AttributeValue, item any) error {
	unmarshaler, ok := item.(Unmarshaler)
	if ok {
		return unmarshaler.UnmarshalDynamo(attrs)
	}
	if err := attributevalue.UnmarshalMap(attrs, item); err != nil {
		return err
	}
	return nil
}
