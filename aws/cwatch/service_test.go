package cwatch_test

import (
	"sync"
	"testing"

	"github.com/nyaruka/gocommon/aws/cwatch"
	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	wg := &sync.WaitGroup{}

	svc, err := cwatch.NewService("root", "key", "us-east-1", "Foo", wg)
	assert.NoError(t, err)

	svc.Start()

	svc.Stop()

	wg.Wait()
}
