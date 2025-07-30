package dynamo

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awsx "github.com/nyaruka/gocommon/aws"
)

// NewClient creates a new DynamoDB client with the provided credentials.
func NewClient(accessKey, secretKey, region, endpoint string) (*dynamodb.Client, error) {
	cfg, err := awsx.NewConfig(accessKey, secretKey, region)
	if err != nil {
		return nil, err
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	return client, nil
}

// Test checks that the given tables exist.
func Test(ctx context.Context, c *dynamodb.Client, tables ...string) error {
	for _, table := range tables {
		_, err := c.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)})
		if err != nil {
			return fmt.Errorf("error describing dynamo table: %w", err)
		}
	}

	return nil
}
