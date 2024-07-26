package s3x

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Service is simple abstraction layer to work with a S3-compatible storage service
type Service struct {
	Client *s3.S3
	urler  ObjectURLer
}

// NewService creates a new S3 service with the given credentials and configuration
func NewService(accessKey, secretKey, region, endpoint string, minio bool) (*Service, error) {
	cfg := &aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(minio), // urls as endpoint/bucket/key instead of bucket.endpoint/key
		MaxRetries:       aws.Int(3),
	}
	if accessKey != "" || secretKey != "" {
		cfg.Credentials = credentials.NewStaticCredentials(accessKey, secretKey, "")
	}
	s, err := session.NewSession(cfg)
	if err != nil {
		return nil, err
	}

	var urler ObjectURLer
	if minio {
		urler = MinioURLer(endpoint)
	} else {
		urler = AWSURLer(region)
	}

	return &Service{Client: s3.New(s), urler: urler}, nil
}

// ObjectURL returns the publicly accessible URL for the given object
func (s *Service) ObjectURL(bucket, key string) string {
	return s.urler(bucket, key)
}

// Test is a convenience method to HEAD a bucket to test if it exists and we can access it
func (s *Service) Test(ctx context.Context, bucket string) error {
	_, err := s.Client.HeadBucket(&s3.HeadBucketInput{Bucket: aws.String(bucket)})
	return err
}

// GetObject is a convenience method to get an object from S3 and read its contents into a byte slice
func (s *Service) GetObject(ctx context.Context, bucket, key string) (string, []byte, error) {
	out, err := s.Client.GetObjectWithContext(ctx, &s3.GetObjectInput{
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

// PutObject is a convenience method to put the given object and return its publicly accessible URL
func (s *Service) PutObject(ctx context.Context, bucket, key string, contentType string, body []byte, acl string) (string, error) {
	_, err := s.Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Body:        bytes.NewReader(body),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		ACL:         aws.String(acl),
	})
	if err != nil {
		return "", fmt.Errorf("error putting S3 object: %w", err)
	}
	return s.urler(bucket, key), nil
}
