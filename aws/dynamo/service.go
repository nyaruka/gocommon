package dynamo

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Service is simple abstraction layer to work with a DynamoDB-compatible database
type Service struct {
	Client      *dynamodb.Client
	tablePrefix string
}

// NewService creates a new dynamodb service with the given credentials and configuration
func NewService(accessKey, secretKey, region, endpoint, tablePrefix string) (*Service, error) {
	opts := []func(*config.LoadOptions) error{config.WithRegion(region)}

	if accessKey != "" && secretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.StaticCredentialsProvider{Value: aws.Credentials{
			AccessKeyID: accessKey, SecretAccessKey: secretKey,
		}}))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		return nil, err
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	return &Service{Client: client, tablePrefix: tablePrefix}, nil
}

// Test checks if the service is working by trying to list tables
func (s *Service) Test(ctx context.Context) error {
	_, err := s.Client.ListTables(ctx, &dynamodb.ListTablesInput{})
	return err
}

// TableName returns the full table name with the prefix
func (s *Service) TableName(base string) string {
	return s.tablePrefix + base
}

// GetItem retrieves an item from the given table
func (s *Service) GetItem(ctx context.Context, table string, key map[string]types.AttributeValue, dst any) error {
	resp, err := s.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.TableName(table)),
		Key:       key,
	})
	if err != nil {
		return fmt.Errorf("error getting item from dynamo: %w", err)
	}

	if err := unmarshal(resp.Item, dst); err != nil {
		return fmt.Errorf("error unmarshaling dynamo item: %w", err)
	}

	return nil
}

// PutItem puts an item into the given table
func (s *Service) PutItem(ctx context.Context, table string, v any) error {
	item, err := marshal(v)
	if err != nil {
		return fmt.Errorf("error marshaling dynamo item: %w", err)
	}

	_, err = s.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.TableName(table)),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("error putting item to dynamo: %w", err)
	}

	return nil
}
