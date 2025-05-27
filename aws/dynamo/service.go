package dynamo

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awsx "github.com/nyaruka/gocommon/aws"
)

// Service is simple abstraction layer to work with a DynamoDB-compatible database
type Service[K, I any] struct {
	Client      *dynamodb.Client
	tablePrefix string
}

// NewService creates a new dynamodb service with the given credentials and configuration
func NewService[K, I any](accessKey, secretKey, region, endpoint, tablePrefix string) (*Service[K, I], error) {
	cfg, err := awsx.NewConfig(accessKey, secretKey, region)
	if err != nil {
		return nil, err
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	return &Service[K, I]{Client: client, tablePrefix: tablePrefix}, nil
}

// Test checks if the service is working by trying to list tables
func (s *Service[K, I]) Test(ctx context.Context) error {
	_, err := s.Client.ListTables(ctx, &dynamodb.ListTablesInput{})
	return err
}

// TableName returns the full table name with the prefix
func (s *Service[K, I]) TableName(base string) string {
	return s.tablePrefix + base
}

// GetItem retrieves an item from the given table
func (s *Service[K, I]) GetItem(ctx context.Context, table string, key K) (*I, error) {
	keyAttrs, _ := attributevalue.MarshalMap(key)

	resp, err := s.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.TableName(table)),
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

// PutItem puts an item into the given table
func (s *Service[K, I]) PutItem(ctx context.Context, table string, item *I) error {
	itemAttrs, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("error marshaling dynamo item: %w", err)
	}

	_, err = s.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.TableName(table)),
		Item:      itemAttrs,
	})
	if err != nil {
		return fmt.Errorf("error putting item in dynamo: %w", err)
	}

	return nil
}
