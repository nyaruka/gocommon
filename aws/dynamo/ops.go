package dynamo

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// GetItem retrieves an item from the table
func GetItem(ctx context.Context, c *dynamodb.Client, table string, key Key) (*Item, error) {
	attrs, err := attributevalue.MarshalMap(key)
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

	item := &Item{}
	if err := attributevalue.UnmarshalMap(resp.Item, item); err != nil {
		return nil, err
	}

	return item, nil
}

// PutItem puts an item into the table
func PutItem(ctx context.Context, c *dynamodb.Client, table string, item *Item) error {
	attrs, err := attributevalue.MarshalMap(item)
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
func BatchPutItem(ctx context.Context, c *dynamodb.Client, table string, items []*Item) ([]*Item, error) {
	marshaled := make([]map[string]types.AttributeValue, len(items))
	for i, item := range items {
		var err error
		marshaled[i], err = attributevalue.MarshalMap(item)
		if err != nil {
			return nil, fmt.Errorf("error marshaling item for batch put: %w", err)
		}
	}

	unprocessedRaw, err := batchPutItem(ctx, c, table, marshaled)
	if err != nil {
		return nil, err
	}

	unprocessed := make([]*Item, len(unprocessedRaw))
	for i, itemAttrs := range unprocessedRaw {
		item := &Item{}
		if err := attributevalue.UnmarshalMap(itemAttrs, item); err != nil {
			return nil, fmt.Errorf("error unmarshaling unprocessed item from batch put: %w", err)
		}
		unprocessed[i] = item
	}

	return unprocessed, nil
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
		return nil, fmt.Errorf("error batch writing items to table %s: %w", table, err)
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
