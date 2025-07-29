package dynamo

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Marshaler interface {
	MarshalDynamo() (map[string]types.AttributeValue, error)
}

type Unmarshaler interface {
	UnmarshalDynamo(map[string]types.AttributeValue) error
}

func Marshal(item any) (map[string]types.AttributeValue, error) {
	marshaler, ok := item.(Marshaler)
	if ok {
		return marshaler.MarshalDynamo()
	}
	return attributevalue.MarshalMap(item)
}

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
