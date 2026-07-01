package dynamo_test

import (
	"os"
	"testing"
)

// TestMain exports the standard AWS_* env vars so the SDK default credential chain resolves against localstack.
func TestMain(m *testing.M) {
	os.Setenv("AWS_ACCESS_KEY_ID", "root")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "tembatemba")
	os.Setenv("AWS_REGION", "us-east-1")

	os.Exit(m.Run())
}
