package s3x

import (
	"fmt"
	"net/url"
)

// ObjectURLer is a function that takes a key and returns the publicly accessible URL for that object
type ObjectURLer func(string) string

func AWSURLer(region, bucket string) ObjectURLer {
	return func(key string) string {
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, url.PathEscape(key))
	}
}

func MinioURLer(endpoint, bucket string) ObjectURLer {
	return func(key string) string {
		return fmt.Sprintf("%s/%s/%s", endpoint, bucket, url.PathEscape(key))
	}
}
