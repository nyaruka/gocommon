package s3x

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Service is simple abstraction layer to work with a S3-compatible storage service
type Service struct {
	client *s3.S3
	urler  ObjectURLer
}

func NewService(client *s3.S3, urler ObjectURLer) *Service {
	return &Service{client: client, urler: urler}
}

func (s *Service) HeadBucket(ctx context.Context, bucket string) error {
	_, err := s.client.HeadBucket(&s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		return fmt.Errorf("error heading bucket: %w", err)
	}
	return nil
}

func (s *Service) CreateBucket(ctx context.Context, bucket string) error {
	_, err := s.client.CreateBucket(&s3.CreateBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		return fmt.Errorf("error creating bucket: %w", err)
	}
	return nil
}

func (s *Service) DeleteBucket(ctx context.Context, bucket string) error {
	_, err := s.client.DeleteBucket(&s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		return fmt.Errorf("error deleting bucket: %w", err)
	}
	return nil
}

func (s *Service) GetObject(ctx context.Context, bucket, key string) (string, []byte, error) {
	out, err := s.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", nil, fmt.Errorf("error getting S3 object: %w", err)
	}

	body, err := io.ReadAll(out.Body)
	if err != nil {
		return "", nil, fmt.Errorf("error reading S3 object: %w", err)
	}

	return aws.StringValue(out.ContentType), body, nil
}

// PutObject writes the passed in file to the given bucket with the passed in content type and ACL
func (s *Service) PutObject(ctx context.Context, bucket, key string, contentType string, body []byte, acl string) (string, error) {
	_, err := s.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Body:        bytes.NewReader(body),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		ACL:         aws.String(acl),
	})
	if err != nil {
		return "", fmt.Errorf("error putting S3 object: %w", err)
	}

	return s.urler(key), nil
}
