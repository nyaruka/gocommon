package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/nyaruka/gocommon/storage"
	"github.com/stretchr/testify/assert"
)

type testS3Client struct {
	returnError           error
	headBucketReturnValue *s3.HeadBucketOutput
	getObjectReturnValue  *s3.GetObjectOutput
	putObjectReturnValue  *s3.PutObjectOutput
}

func (c *testS3Client) HeadBucketWithContext(ctx context.Context, input *s3.HeadBucketInput, opts ...request.Option) (*s3.HeadBucketOutput, error) {
	if c.returnError != nil {
		return nil, c.returnError
	}
	return c.headBucketReturnValue, nil
}
func (c *testS3Client) GetObjectWithContext(ctx context.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error) {
	if c.returnError != nil {
		return nil, c.returnError
	}
	return c.getObjectReturnValue, nil
}
func (c *testS3Client) PutObjectWithContext(ctx context.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	if c.returnError != nil {
		return nil, c.returnError
	}
	return c.putObjectReturnValue, nil
}

func TestS3Test(t *testing.T) {
	client := &testS3Client{}
	s3 := storage.NewS3(client, "mybucket", "us-east-1", 1)

	assert.NoError(t, s3.Test(context.Background()))

	client.returnError = errors.New("boom")

	assert.EqualError(t, s3.Test(context.Background()), "boom")
}

func TestS3Get(t *testing.T) {
	ctx := context.Background()
	client := &testS3Client{}
	s := storage.NewS3(client, "mybucket", "us-east-1", 1)

	client.getObjectReturnValue = &s3.GetObjectOutput{
		ContentType: aws.String("text/plain"),
		Body:        io.NopCloser(bytes.NewReader([]byte(`HELLOWORLD`))),
	}

	contentType, contents, err := s.Get(ctx, "/foo/things")
	assert.NoError(t, err)
	assert.Equal(t, "text/plain", contentType)
	assert.Equal(t, []byte(`HELLOWORLD`), contents)

	client.returnError = errors.New("boom")

	_, _, err = s.Get(ctx, "/foo/things")
	assert.EqualError(t, err, "error getting S3 object: boom")
}

func TestS3Put(t *testing.T) {
	ctx := context.Background()
	client := &testS3Client{}
	s := storage.NewS3(client, "mybucket", "us-east-1", 1)

	url, err := s.Put(ctx, "/foo/things", "text/plain", []byte(`HELLOWORLD`))
	assert.NoError(t, err)
	assert.Equal(t, "https://mybucket.s3.us-east-1.amazonaws.com/foo/things", url)

	client.returnError = errors.New("boom")

	_, err = s.Put(ctx, "/foo/things", "text/plain", []byte(`HELLOWORLD`))
	assert.EqualError(t, err, "error putting S3 object: boom")
}

func TestS3BatchPut(t *testing.T) {

	ctx := context.Background()
	client := &testS3Client{}
	s := storage.NewS3(client, "mybucket", "us-east-1", 10)

	uploads := []*storage.Upload{
		{
			Path:        "https://mybucket.s3.us-east-1.amazonaws.com/foo/thing1",
			Body:        []byte(`HELLOWORLD`),
			ContentType: "text/plain",
			ACL:         s3.BucketCannedACLPrivate,
		},
		{
			Path:        "https://mybucket.s3.us-east-1.amazonaws.com/foo/thing2",
			Body:        []byte(`HELLOWORLD2`),
			ContentType: "text/plain",
			ACL:         s3.BucketCannedACLPrivate,
		},
	}

	err := s.BatchPut(ctx, uploads)
	assert.NoError(t, err)

	assert.NotEmpty(t, uploads[0].URL)
	assert.NotEmpty(t, uploads[1].URL)

	// try again, with a single thread and throwing an error
	s = storage.NewS3(client, "mybucket", "us-east-1", 1)
	client.returnError = errors.New("boom")

	uploads[0].URL = ""
	uploads[1].URL = ""

	err = s.BatchPut(ctx, uploads)

	assert.Error(t, err)

	assert.Empty(t, uploads[0].URL)
	assert.Empty(t, uploads[1].URL)
	assert.NotEmpty(t, uploads[0].Error)
}
