// DynamoDB utility functions which should never be used in production but are useful for testing. Thus they are gated
// to only work with tables that start with "Test" prefix.
package dyntest

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/jsonx"
	"github.com/stretchr/testify/require"
)

func CreateTables(t *testing.T, c *dynamodb.Client, path string, delExisting bool) {
	t.Helper()
	ctx := t.Context()

	tablesFile, err := os.Open(path)
	require.NoError(t, err)
	defer tablesFile.Close()

	tablesJSON, err := io.ReadAll(tablesFile)
	require.NoError(t, err)

	inputs := []*dynamodb.CreateTableInput{}
	jsonx.MustUnmarshal(tablesJSON, &inputs)

	for _, input := range inputs {
		input.TableName = aws.String("Test" + *input.TableName) // add "Test" prefix

		_, err := c.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: input.TableName})
		exists := err == nil

		// delete table if it exists
		if exists && delExisting {
			_, err := c.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: input.TableName})
			require.NoError(t, err)

			exists = false
		}

		if !exists {
			_, err := c.CreateTable(ctx, input)
			require.NoError(t, err)
		}
	}
}

// Truncate deletes all items in the table
func Truncate(t *testing.T, c *dynamodb.Client, table string) {
	t.Helper()
	assertTesting(t, table)
	ctx := t.Context()

	desc, err := c.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(table),
	})
	require.NoError(t, err)

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
		require.NoError(t, err)

		for _, item := range output.Items {
			_, err := c.DeleteItem(ctx, &dynamodb.DeleteItemInput{
				TableName: aws.String(table),
				Key:       item,
			})
			require.NoError(t, err)
		}

		lastEvaluatedKey = output.LastEvaluatedKey
		if lastEvaluatedKey == nil {
			break // no more items to delete
		}
	}
}

// Drop deletes entire tables.
func Drop(t *testing.T, c *dynamodb.Client, tables ...string) {
	t.Helper()
	ctx := t.Context()

	for _, table := range tables {
		assertTesting(t, table)

		_, err := c.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(table)})
		require.NoError(t, err)
	}
}
