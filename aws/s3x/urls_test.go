package s3x_test

import (
	"testing"

	"github.com/nyaruka/gocommon/aws/s3x"
	"github.com/stretchr/testify/assert"
)

func TestURLers(t *testing.T) {
	urler := s3x.VirtualHostURLer("us-east-1")
	assert.Equal(t, "https://mybucket.s3.us-east-1.amazonaws.com/1/hello+world.txt", urler("mybucket", "1/hello world.txt"))

	urler = s3x.PathStyleURLer("http://localstack:4566")
	assert.Equal(t, "http://localstack:4566/mybucket/1/hello+world.txt", urler("mybucket", "1/hello world.txt"))
}
