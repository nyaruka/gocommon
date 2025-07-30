package dyntest

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertCount asserts the total number of items in a table
func AssertCount(t *testing.T, c *dynamodb.Client, table string, expected int, msgAndArgs ...any) bool {
	t.Helper()
	ctx := context.Background()
	assertTesting(table)

	output, err := c.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(table),
		Select:    "COUNT",
	})
	require.NoError(t, err)

	return assert.Equal(t, expected, int(output.Count), msgAndArgs...)
}

func assertTesting(table string) {
	if !strings.HasPrefix(table, "Test") {
		panic("can only be called on table named with 'Test' prefix")
	}
}
