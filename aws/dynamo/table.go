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

// NewTable creates a new dynamodb table
func NewTable[K, I any](client *dynamodb.Client, name string) *Table[K, I] {
	return &Table[K, I]{Client: client, name: name}
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

// Count returns the number of items in the table.. for testing purposes
func (t *Table[K, I]) Count(ctx context.Context) (int, error) {
	output, err := t.Client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(t.name),
		Select:    "COUNT",
	})
	if err != nil {
		return 0, fmt.Errorf("error scanning table for count: %w", err)
	}

	return int(output.Count), nil
}

// Delete deletes the entire table
func (t *Table[K, I]) Delete(ctx context.Context) error {
	_, err := t.Client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(t.name)})
	return err
}
