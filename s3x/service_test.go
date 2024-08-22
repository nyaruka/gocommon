package s3x_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/nyaruka/gocommon/s3x"
	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	ctx := context.Background()

	svc, err := s3x.NewService("root", "tembatemba", "us-east-1", "http://localhost:9000", true)
	assert.NoError(t, err)

	err = svc.Test(ctx, "gocommon-tests")
	assert.ErrorContains(t, err, "NotFound")

	_, err = svc.Client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("gocommon-tests")})
	assert.NoError(t, err)

	err = svc.Test(ctx, "gocommon-tests")
	assert.NoError(t, err)

	url, err := svc.PutObject(ctx, "gocommon-tests", "1/hello world.txt", "text/plain", []byte("hello world"), types.ObjectCannedACLPublicRead)
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:9000/gocommon-tests/1/hello+world.txt", url)

	contentType, body, err := svc.GetObject(ctx, "gocommon-tests", "1/hello world.txt")
	assert.NoError(t, err)
	assert.Equal(t, "text/plain", contentType)
	assert.Equal(t, []byte("hello world"), body)

	// test batch put
	uploads := []*s3x.Upload{
		{
			Bucket:      "gocommon-tests",
			Key:         "foo/thing1",
			Body:        []byte(`HELLOWORLD`),
			ContentType: "text/plain",
			ACL:         types.ObjectCannedACLPublicRead,
		},
		{
			Bucket:      "gocommon-tests",
			Key:         "foo/thing2",
			Body:        []byte(`HELLOWORLD2`),
			ContentType: "text/plain",
			ACL:         types.ObjectCannedACLPublicRead,
		},
	}

	err = svc.BatchPut(ctx, uploads, 3)
	assert.NoError(t, err)

	assert.Equal(t, "http://localhost:9000/gocommon-tests/foo/thing1", uploads[0].URL)
	assert.Nil(t, uploads[0].Error)
	assert.Equal(t, "http://localhost:9000/gocommon-tests/foo/thing2", uploads[1].URL)
	assert.Nil(t, uploads[1].Error)

	// test emptying a bucket
	err = svc.EmptyBucket(ctx, "gocommon-tests")
	assert.NoError(t, err)

	err = svc.EmptyBucket(ctx, "gocommon-tests")
	assert.NoError(t, err)

	// test deleting a bucket
	_, err = svc.Client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String("gocommon-tests")})
	assert.NoError(t, err)

	err = svc.Test(ctx, "gocommon-tests")
	assert.Error(t, err)

	aws, err := s3x.NewService("AA1234", "2345263", "us-east-1", "https://s3.amazonaws.com", false)
	assert.NoError(t, err)
	assert.Equal(t, "https://gocommon-tests.s3.us-east-1.amazonaws.com/1/hello+world.txt", aws.ObjectURL("gocommon-tests", "1/hello world.txt"))
}
