package s3x_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/nyaruka/gocommon/s3x"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAndPutObject(t *testing.T) {
	ctx := context.Background()

	config := &aws.Config{
		Endpoint:         aws.String("http://localhost:9000"),
		Region:           aws.String("us-east-1"),
		Credentials:      credentials.NewStaticCredentials("root", "tembatemba", ""),
		S3ForcePathStyle: aws.Bool(true),
	}
	s, err := session.NewSession(config)
	require.NoError(t, err)

	client := s3.New(s)
	require.NotNil(t, client)

	svc := s3x.NewService(client, s3x.MinioURLer("http://localhost:9000", "mybucket"))

	err = svc.HeadBucket(ctx, "gocommon-tests")
	assert.ErrorContains(t, err, "error heading bucket: NotFound: Not Found\n\tstatus code: 404")

	err = svc.CreateBucket(ctx, "gocommon-tests")
	assert.NoError(t, err)

	err = svc.HeadBucket(ctx, "gocommon-tests")
	assert.NoError(t, err)

	url, err := svc.PutObject(ctx, "gocommon-tests", "test.txt", "text/plain", []byte("hello world"), s3.BucketCannedACLPublicRead)
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:9000/mybucket/test.txt", url)

	contentType, body, err := svc.GetObject(ctx, "gocommon-tests", "test.txt")
	assert.NoError(t, err)
	assert.Equal(t, "text/plain", contentType)
	assert.Equal(t, []byte("hello world"), body)

	_, err = client.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String("gocommon-tests"), Key: aws.String("test.txt")})
	assert.NoError(t, err)

	err = svc.DeleteBucket(ctx, "gocommon-tests")
	assert.NoError(t, err)

	err = svc.HeadBucket(ctx, "gocommon-tests")
	assert.Error(t, err)
}
