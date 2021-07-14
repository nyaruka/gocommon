package storage

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

var s3BucketURL = "https://%s.s3.%s.amazonaws.com%s"

// S3Client provides a mockable subset of the S3 API
type S3Client interface {
	HeadBucketWithContext(ctx context.Context, input *s3.HeadBucketInput, opts ...request.Option) (*s3.HeadBucketOutput, error)
	GetObjectWithContext(ctx context.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error)
	PutObjectWithContext(ctx context.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error)
}

// S3Options are options for an S3 client
type S3Options struct {
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	Endpoint           string
	Region             string
	DisableSSL         bool
	ForcePathStyle     bool
	WorkersPerBatch    int
}

// NewS3Client creates a new S3 client
func NewS3Client(opts *S3Options) (S3Client, error) {
	s3Session, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(opts.AWSAccessKeyID, opts.AWSSecretAccessKey, ""),
		Endpoint:         aws.String(opts.Endpoint),
		Region:           aws.String(opts.Region),
		DisableSSL:       aws.Bool(opts.DisableSSL),
		S3ForcePathStyle: aws.Bool(opts.ForcePathStyle),
	})
	if err != nil {
		return nil, err
	}

	return s3.New(s3Session), nil
}

type s3Storage struct {
	client          S3Client
	bucket          string
	region          string
	workersPerBatch int
}

// NewS3 creates a new S3 storage service. Callers can specify how many parallel uploads will take place at
// once when calling BatchPut with workersPerBatch
func NewS3(client S3Client, bucket, region string, workersPerBatch int) Storage {
	return &s3Storage{client: client, bucket: bucket, region: region, workersPerBatch: workersPerBatch}
}

func (s *s3Storage) Name() string {
	return "S3"
}

// Test tests whether our S3 client is properly configured
func (s *s3Storage) Test(ctx context.Context) error {
	_, err := s.client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	return err
}

func (s *s3Storage) Get(ctx context.Context, path string) (string, []byte, error) {
	out, err := s.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return "", nil, errors.Wrapf(err, "error getting S3 object")
	}

	contents, err := ioutil.ReadAll(out.Body)
	if err != nil {
		return "", nil, errors.Wrapf(err, "error reading S3 object")
	}

	return aws.StringValue(out.ContentType), contents, nil
}

// Put writes the passed in file to the bucket with the passed in content type
func (s *s3Storage) Put(ctx context.Context, path string, contentType string, contents []byte) (string, error) {
	_, err := s.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Body:        bytes.NewReader(contents),
		Key:         aws.String(path),
		ContentType: aws.String(contentType),
		ACL:         aws.String(s3.BucketCannedACLPublicRead),
	})
	if err != nil {
		return "", errors.Wrapf(err, "error putting S3 object")
	}

	return s.url(path), nil
}

func (s *s3Storage) batchWorker(ctx context.Context, uploads chan *Upload, errors chan error, stop chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
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

				_, err = s.client.PutObjectWithContext(uctx, &s3.PutObjectInput{
					Bucket:      aws.String(s.bucket),
					Body:        bytes.NewReader(u.Body),
					Key:         aws.String(u.Path),
					ContentType: aws.String(u.ContentType),
					ACL:         aws.String(u.ACL),
				})

				if err == nil {
					break
				}
			}

			if err == nil {
				u.URL = s.url(u.Path)
			} else {
				u.Error = err
			}

			errors <- err

		case <-stop:
			return
		}
	}
}

// BatchPut writes the entire batch of items to the passed in URLs, returning a map of errors if any.
// Writes will be retried up to three times automatically.
func (s *s3Storage) BatchPut(ctx context.Context, us []*Upload) error {
	uploads := make(chan *Upload, len(us))
	errors := make(chan error, len(us))
	stop := make(chan bool)
	wg := &sync.WaitGroup{}

	// start our workers
	for w := 0; w < s.workersPerBatch; w++ {
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

func (s *s3Storage) url(path string) string {
	return fmt.Sprintf(s3BucketURL, s.bucket, s.region, path)
}
