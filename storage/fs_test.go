package storage_test

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/nyaruka/gocommon/storage"
	"github.com/nyaruka/gocommon/uuids"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFS(t *testing.T) {
	ctx := context.Background()
	uuids.SetGenerator(uuids.NewSeededGenerator(12345))
	defer uuids.SetGenerator(uuids.DefaultGenerator)

	s := storage.NewFS("_testing")
	assert.NoError(t, s.Test(ctx))

	// break our ability to write to that directory
	require.NoError(t, os.Chmod("_testing", 0555))

	assert.EqualError(t, s.Test(ctx), "open _testing/e7187099-7d38-4f60-955c-325957214c42.txt: permission denied")

	require.NoError(t, os.Chmod("_testing", 0777))

	url, err := s.Put(ctx, "/foo/bar.txt", "text/plain", []byte(`hello world`))
	assert.NoError(t, err)
	assert.Equal(t, "_testing/foo/bar.txt", url)

	_, data, err := s.Get(ctx, "/foo/bar.txt")
	assert.NoError(t, err)
	assert.Equal(t, []byte(`hello world`), data)

	require.NoError(t, os.RemoveAll("_testing"))
}

func TestFSBatchPut(t *testing.T) {

	ctx := context.Background()
	s := storage.NewFS("_testing")

	uploads := []*storage.Upload{
		&storage.Upload{
			Path:        "https://mybucket.s3.amazonaws.com/foo/thing1",
			Body:        []byte(`HELLOWORLD`),
			ContentType: "text/plain",
			ACL:         s3.BucketCannedACLPrivate,
		},
		&storage.Upload{
			Path:        "https://mybucket.s3.amazonaws.com/foo/thing2",
			Body:        []byte(`HELLOWORLD2`),
			ContentType: "text/plain",
			ACL:         s3.BucketCannedACLPrivate,
		},
	}

	// no writing to our test dir, will fail
	require.NoError(t, os.Chmod("_testing", 0555))

	err := s.BatchPut(ctx, uploads)
	assert.Error(t, err)

	assert.Empty(t, uploads[0].URL)
	assert.Empty(t, uploads[1].URL)
	assert.NotEmpty(t, uploads[0].Error)

	// fix dir permissions, try again
	require.NoError(t, os.Chmod("_testing", 0777))

	err = s.BatchPut(ctx, uploads)
	assert.NoError(t, err)

	assert.NotEmpty(t, uploads[0].URL)
	assert.NotEmpty(t, uploads[1].URL)
}
