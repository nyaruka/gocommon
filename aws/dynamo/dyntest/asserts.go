package dyntest

import (
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
	assertTesting(t, table)

	output, err := c.Scan(t.Context(), &dynamodb.ScanInput{
		TableName: aws.String(table),
		Select:    "COUNT",
	})
	require.NoError(t, err)

	return assert.Equal(t, expected, int(output.Count), msgAndArgs...)
}

func assertTesting(t *testing.T, table string) {
	require.True(t, strings.HasPrefix(table, "Test"), "table name must start with 'Test' prefix")
}
