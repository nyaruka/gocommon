package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// NewConfig creates a new AWS config with the given credentials and region
func NewConfig(accessKey, secretKey, region string) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{config.WithRegion(region)}

	if accessKey != "" && secretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.StaticCredentialsProvider{Value: aws.Credentials{
			AccessKeyID: accessKey, SecretAccessKey: secretKey,
		}}))
	}

	return config.LoadDefaultConfig(context.TODO(), opts...)
}
