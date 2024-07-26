package s3x_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/nyaruka/gocommon/s3x"
	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	ctx := context.Background()

	svc, err := s3x.NewService("root", "tembatemba", "us-east-1", "http://localhost:9000", true)
	assert.NoError(t, err)

	err = svc.Test(ctx, "gocommon-tests")
	assert.ErrorContains(t, err, "NotFound: Not Found\n\tstatus code: 404")

	_, err = svc.Client.CreateBucket(&s3.CreateBucketInput{Bucket: aws.String("gocommon-tests")})
	assert.NoError(t, err)

	err = svc.Test(ctx, "gocommon-tests")
	assert.NoError(t, err)

	url, err := svc.PutObject(ctx, "gocommon-tests", "1/hello world.txt", "text/plain", []byte("hello world"), s3.BucketCannedACLPublicRead)
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:9000/gocommon-tests/1/hello+world.txt", url)

	contentType, body, err := svc.GetObject(ctx, "gocommon-tests", "1/hello world.txt")
	assert.NoError(t, err)
	assert.Equal(t, "text/plain", contentType)
	assert.Equal(t, []byte("hello world"), body)

	_, err = svc.Client.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String("gocommon-tests"), Key: aws.String("1/hello world.txt")})
	assert.NoError(t, err)

	_, err = svc.Client.DeleteBucket(&s3.DeleteBucketInput{Bucket: aws.String("gocommon-tests")})
	assert.NoError(t, err)

	err = svc.Test(ctx, "gocommon-tests")
	assert.Error(t, err)

	aws, err := s3x.NewService("AA1234", "2345263", "us-east-1", "https://s3.amazonaws.com", false)
	assert.NoError(t, err)
	assert.Equal(t, "https://gocommon-tests.s3.us-east-1.amazonaws.com/1/hello+world.txt", aws.ObjectURL("gocommon-tests", "1/hello world.txt"))
}
