package s3x

import (
	"fmt"
	"strings"
)

// ObjectURLer is a function that takes a key and returns the publicly accessible URL for that object
type ObjectURLer func(string, string) string

func AWSURLer(region string) ObjectURLer {
	return func(bucket, key string) string {
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, escape(key))
	}
}

func MinioURLer(endpoint string) ObjectURLer {
	return func(bucket, key string) string {
		return fmt.Sprintf("%s/%s/%s", endpoint, bucket, escape(key))
	}
}

// can't URL escape keys because need to preserve slashes
func escape(key string) string {
	return strings.ReplaceAll(key, " ", "+")
}
