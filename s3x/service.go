package s3x

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

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

// Upload is our type for a file in a batch upload
type Upload struct {
	Bucket      string
	Key         string
	ContentType string
	Body        []byte
	ACL         string

	// set by BatchPut
	URL   string
	Error error
}

// BatchPut writes the entire batch of items to the passed in URLs, returning a map of errors if any.
// Writes will be retried up to three times automatically.
func (s *Service) BatchPut(ctx context.Context, us []*Upload, workers int) error {
	uploads := make(chan *Upload, len(us))
	errors := make(chan error, len(us))
	stop := make(chan bool)
	wg := &sync.WaitGroup{}

	// start our workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go s.batchWorker(ctx, uploads, errors, stop, wg)
	}

	// add all our uploads to our work queue
	for _, u := range us {
		uploads <- u
	}

	// read all our errors out, we'll stop everything if we encounter one
	var err error
	for i := 0; i < len(us); i++ {
		e := <-errors
		if e != nil {
			err = e
			break
		}
	}

	// stop everyone
	close(stop)

	// wait for everything to finish up
	wg.Wait()

	return err
}

func (s *Service) batchWorker(ctx context.Context, uploads chan *Upload, errors chan error, stop chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case u := <-uploads:
			var err error
			for tries := 0; tries < 3; tries++ {
				// we use a short timeout per request, better to retry than wait on a stalled connection and waste all our time
				// TODO: validate choice of 15 seconds against real world performance
				uctx, cancel := context.WithTimeout(ctx, time.Second*15)
				defer cancel()

				_, err = s.Client.PutObjectWithContext(uctx, &s3.PutObjectInput{
					Bucket:      aws.String(u.Bucket),
					Body:        bytes.NewReader(u.Body),
					Key:         aws.String(u.Key),
					ContentType: aws.String(u.ContentType),
					ACL:         aws.String(u.ACL),
				})

				if err == nil {
					break
				}
			}

			if err == nil {
				u.URL = s.urler(u.Bucket, u.Key)
			} else {
				u.Error = err
			}

			errors <- err

		case <-stop:
			return
		}
	}
}

// EmptyBucket is a convenience method to delete all the objects in a bucket
func (s *Service) EmptyBucket(ctx context.Context, bucket string) error {
	request := &s3.ListObjectsV2Input{Bucket: aws.String(bucket)}

	for {
		response, err := s.Client.ListObjectsV2WithContext(ctx, request)
		if err != nil {
			return fmt.Errorf("error listing S3 objects: %w", err)
		}

		if len(response.Contents) > 0 {
			del := &s3.Delete{}

			for _, obj := range response.Contents {
				del.Objects = append(del.Objects, &s3.ObjectIdentifier{Key: obj.Key})
			}

			_, err = s.Client.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{Bucket: aws.String(bucket), Delete: del})
			if err != nil {
				return fmt.Errorf("error deleting S3 objects: %w", err)
			}
		}

		request.ContinuationToken = response.NextContinuationToken

		if !aws.BoolValue(response.IsTruncated) {
			break
		}
	}

	return nil
}
