package dynamo

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Service is simple abstraction layer to work with a DynamoDB-compatible database
type Service struct {
	Client      *dynamodb.Client
	tablePrefix string
}

// NewService creates a new S3 service with the given credentials and configuration
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

func (s *Service) Test(ctx context.Context, table string) error {
	_, err := s.Client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(s.TableName(table))})
	return err
}

func (s *Service) TableName(base string) string {
	return s.tablePrefix + base
}
