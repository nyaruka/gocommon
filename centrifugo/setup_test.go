package centrifugo_test

import (
	"os"
	"testing"

	"github.com/nyaruka/vkutil/assertvk"
)

// TestMain opts into coordinated test database claims, matching the dev stack's valkey.
func TestMain(m *testing.M) {
	assertvk.Coordinate(16, 17, 63)

	os.Exit(m.Run())
}
