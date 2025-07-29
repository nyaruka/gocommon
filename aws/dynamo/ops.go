package dynamo

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// GetItem retrieves an item from the table
func GetItem[K, I any](ctx context.Context, c *dynamodb.Client, table string, key K) (*I, error) {
	attrs, err := Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("error marshaling key for get: %w", err)
	}

	resp, err := c.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(table),
		Key:       attrs,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting item from dynamo: %w", err)
	}

	if resp.Item == nil {
		return nil, nil // item not found
	}

	item := new(I)
	if err := Unmarshal(resp.Item, item); err != nil {
		return nil, err
	}

	return item, nil
}

// PutItem puts an item into the table
func PutItem[I any](ctx context.Context, c *dynamodb.Client, table string, item *I) error {
	attrs, err := Marshal(item)
	if err != nil {
		return fmt.Errorf("error marshaling item for put: %w", err)
	}

	_, err = c.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      attrs,
	})
	if err != nil {
		return fmt.Errorf("error putting item to dynamo: %w", err)
	}

	return nil
}

// BatchPutItem puts multiple items into the table (max 25 items)
func BatchPutItem[I any](ctx context.Context, c *dynamodb.Client, table string, items []*I) ([]map[string]types.AttributeValue, error) {
	marshaled := make([]map[string]types.AttributeValue, len(items))
	for i, item := range items {
		var err error
		marshaled[i], err = Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("error marshaling item for batch put: %w", err)
		}
	}

	return batchPutItem(ctx, c, table, marshaled)
}

// puts multiple items into the table (max 25 items)
func batchPutItem(ctx context.Context, c *dynamodb.Client, table string, items []map[string]types.AttributeValue) ([]map[string]types.AttributeValue, error) {
	if len(items) == 0 {
		return nil, nil
	}

	writeRequests := make([]types.WriteRequest, 0, len(items))

	for _, item := range items {
		writeRequests = append(writeRequests, types.WriteRequest{PutRequest: &types.PutRequest{Item: item}})
	}

	resp, err := c.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			table: writeRequests,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error batch writing items to dynamo: %w", err)
	}

	var unprocessed []map[string]types.AttributeValue

	if unprocessedRequests, exists := resp.UnprocessedItems[table]; exists {
		unprocessed = make([]map[string]types.AttributeValue, 0, len(unprocessedRequests))

		for _, req := range unprocessedRequests {
			unprocessed = append(unprocessed, req.PutRequest.Item)
		}
	}

	return unprocessed, nil
}

// Test checks if the given table exists
func Test(ctx context.Context, c *dynamodb.Client, table string) error {
	_, err := c.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)})
	if err != nil {
		return fmt.Errorf("error describing dynamo table: %w", err)
	}

	return nil
}
