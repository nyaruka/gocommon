// DynamoDB utility functions which should never be used in production but are useful for testing. Thus they are gated
// to only work with tables that start with "Test" prefix.
package dynamo

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Count returns the number of items in a table
func Count(ctx context.Context, c *dynamodb.Client, table string) (int, error) {
	assertTesting(table)

	output, err := c.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(table),
		Select:    "COUNT",
	})
	if err != nil {
		return 0, fmt.Errorf("error scanning table for count: %w", err)
	}

	return int(output.Count), nil
}

// Purge deletes all items in the table
func Purge(ctx context.Context, c *dynamodb.Client, table string) error {
	assertTesting(table)

	desc, err := c.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(table),
	})
	if err != nil {
		return fmt.Errorf("error describing table for purge: %w", err)
	}

	var keyAttrs []string
	for _, attr := range desc.Table.KeySchema {
		keyAttrs = append(keyAttrs, aws.ToString(attr.AttributeName))
	}

	var lastEvaluatedKey map[string]types.AttributeValue
	for {
		output, err := c.Scan(ctx, &dynamodb.ScanInput{
			TableName:            aws.String(table),
			ExclusiveStartKey:    lastEvaluatedKey,
			ProjectionExpression: aws.String(strings.Join(keyAttrs, ",")),
		})
		if err != nil {
			return fmt.Errorf("error scanning table for purge: %w", err)
		}

		for _, item := range output.Items {
			_, err := c.DeleteItem(ctx, &dynamodb.DeleteItemInput{
				TableName: aws.String(table),
				Key:       item,
			})
			if err != nil {
				return fmt.Errorf("error deleting item during purge: %w", err)
			}
		}

		lastEvaluatedKey = output.LastEvaluatedKey
		if lastEvaluatedKey == nil {
			break // no more items to delete
		}
	}

	return nil
}

// Delete deletes the entire table.. for testing purposes
func Delete(ctx context.Context, c *dynamodb.Client, table string) error {
	assertTesting(table)

	_, err := c.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(table)})
	return err
}

func assertTesting(table string) {
	if !strings.HasPrefix(table, "Test") {
		panic("can only be called on table named with 'Test' prefix")
	}
}
