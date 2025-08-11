package dyntest_test

import (
	"testing"

	"github.com/nyaruka/gocommon/aws/dynamo"
	"github.com/nyaruka/gocommon/aws/dynamo/dyntest"
	"github.com/stretchr/testify/assert"
)

type ThingKey struct {
	PK string `dynamodbav:"PK"`
	SK string `dynamodbav:"SK"`
}

type ThingItem struct {
	ThingKey

	Name  string `dynamodbav:"Name"`
	Count int    `dynamodbav:"Count"`
}

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

	thing1 := &ThingItem{ThingKey: ThingKey{PK: "P11", SK: "SAA"}, Name: "Thing 1", Count: 42}
	thing2 := &ThingItem{ThingKey: ThingKey{PK: "P22", SK: "SBB"}, Name: "Thing 2", Count: 235}

	err = dynamo.PutItem(ctx, client, "TestThings", thing1)
	assert.NoError(t, err)
	err = dynamo.PutItem(ctx, client, "TestThings", thing2)
	assert.NoError(t, err)

	dyntest.AssertCount(t, client, "TestThings", 2)
	dyntest.AssertCount(t, client, "TestHistory", 0)

	items := dyntest.ScanAll[ThingItem](t, client, "TestThings")
	assert.ElementsMatch(t, []*ThingItem{thing1, thing2}, items)

	dyntest.Truncate(t, client, "TestThings")

	dyntest.AssertCount(t, client, "TestThings", 0)
	dyntest.AssertCount(t, client, "TestHistory", 0)

	dyntest.Drop(t, client, "TestThings", "TestHistory")

	err = dynamo.Test(ctx, client, "TestThings")
	assert.Error(t, err)
	err = dynamo.Test(ctx, client, "TestHistory")
	assert.Error(t, err)
}
