package dyntest_test

import (
	"testing"

	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/nyaruka/gocommon/aws/dynamo/dyntest"
	"github.com/stretchr/testify/assert"
)

func TestOps(t *testing.T) {
	ctx := t.Context()

	client, err := dynamo.NewClient("root", "tembatemba", "us-east-1", "http://localhost:6000")
	assert.NoError(t, err)

	dyntest.CreateTables(t, client, "./testdata/tables.json", true)
	dyntest.CreateTables(t, client, "./testdata/tables.json", false)

	err = dynamo.Test(ctx, client, "TestThings", "TestHistory")
	assert.NoError(t, err)

	dyntest.AssertCount(t, client, "TestThings", 0)
	dyntest.AssertCount(t, client, "TestHistory", 0)

	thing1 := &dynamo.Item{Key: dynamo.Key{PK: "P11", SK: "SAA"}, OrgID: 1, Data: map[string]any{"name": "Thing 1"}}
	thing2 := &dynamo.Item{Key: dynamo.Key{PK: "P22", SK: "SBB"}, OrgID: 1, Data: map[string]any{"name": "Thing 2"}}

	err = dynamo.PutItem(ctx, client, "TestThings", thing1)
	assert.NoError(t, err)
	err = dynamo.PutItem(ctx, client, "TestThings", thing2)
	assert.NoError(t, err)

	dyntest.AssertCount(t, client, "TestThings", 2)
	dyntest.AssertCount(t, client, "TestHistory", 0)

	items := dyntest.ScanAll(t, client, "TestThings")
	assert.ElementsMatch(t, []*dynamo.Item{thing1, thing2}, items)

	dyntest.Truncate(t, client, "TestThings")

	dyntest.AssertCount(t, client, "TestThings", 0)
	dyntest.AssertCount(t, client, "TestHistory", 0)

	dyntest.Drop(t, client, "TestThings", "TestHistory")

	err = dynamo.Test(ctx, client, "TestThings")
	assert.Error(t, err)
	err = dynamo.Test(ctx, client, "TestHistory")
	assert.Error(t, err)
}
