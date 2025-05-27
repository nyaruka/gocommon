package dynamo

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Table is abstraction layer to work with a DynamoDB-compatible table
type Table[K, I any] struct {
	Client *dynamodb.Client
	name   string
}

// NewTable creates a new dynamodb table with the given credentials and configuration
func NewTable[K, I any](accessKey, secretKey, region, endpoint, name string) (*Table[K, I], error) {
	client, err := NewClient(accessKey, secretKey, region, endpoint)
	if err != nil {
		return nil, err
	}

	return &Table[K, I]{Client: client, name: name}, nil
}

// Name returns the name of the table
func (t *Table[K, I]) Name() string { return t.name }

// Test checks if the service is working by trying to describe the table
func (t *Table[K, I]) Test(ctx context.Context) error {
	_, err := t.Client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(t.name)})
	return err
}

// GetItem retrieves an item from the table
func (t *Table[K, I]) GetItem(ctx context.Context, key K) (*I, error) {
	keyAttrs, _ := attributevalue.MarshalMap(key)

	resp, err := t.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(t.name),
		Key:       keyAttrs,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting item from dynamo: %w", err)
	}

	if resp.Item == nil {
		return nil, nil // item not found
	}

	item := new(I)
	if err := attributevalue.UnmarshalMap(resp.Item, &item); err != nil {
		return nil, fmt.Errorf("error unmarshalling dynamo item: %w", err)
	}

	return item, nil
}

// PutItem puts an item into the table
func (t *Table[K, I]) PutItem(ctx context.Context, item *I) error {
	itemAttrs, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("error marshaling dynamo item: %w", err)
	}

	_, err = t.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(t.name),
		Item:      itemAttrs,
	})
	if err != nil {
		return fmt.Errorf("error putting item in dynamo: %w", err)
	}

	return nil
}

// Delete deletes the entire table
func (t *Table[K, I]) Delete(ctx context.Context) error {
	_, err := t.Client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(t.name)})
	return err
}
