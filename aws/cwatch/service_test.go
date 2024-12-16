package cwatch_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/nyaruka/gocommon/aws/cwatch"
	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	// create service for test environment
	svc, err := cwatch.NewService("root", "key", "us-east-1", "Foo", "test")
	assert.NoError(t, err)

	err = svc.Send(context.Background(), types.MetricDatum{MetricName: aws.String("NumSheep"), Dimensions: []types.Dimension{{Name: aws.String("Host"), Value: aws.String("foo1")}}, Value: aws.Float64(20)})
	assert.NoError(t, err)
	assert.Equal(t, 1, svc.Client.(*cwatch.DevClient).CallCount())

	// check Queue sends synchronously
	svc.Queue(types.MetricDatum{MetricName: aws.String("NumGoats"), Value: aws.Float64(10), Unit: types.StandardUnitCount})
	assert.Equal(t, 2, svc.Client.(*cwatch.DevClient).CallCount())

	// create service for dev environment
	svc, err = cwatch.NewService("root", "key", "us-east-1", "Foo", "dev")
	assert.NoError(t, err)

	svc.StartQueue(time.Millisecond * 100)

	svc.Queue(cwatch.Datum("NumGoats", 10, types.StandardUnitCount, cwatch.Dimension("Host", "foo1")))
	svc.Queue(cwatch.Datum("NumSheep", 20, types.StandardUnitCount))
	assert.Equal(t, 0, svc.Client.(*cwatch.DevClient).CallCount()) // not sent yet

	time.Sleep(time.Millisecond * 200)

	assert.Equal(t, 1, svc.Client.(*cwatch.DevClient).CallCount()) // sent as one call

	svc.Queue(cwatch.Datum("SleepTime", 30, types.StandardUnitSeconds))

	svc.StopQueue()

	// check the queued metric was sent
	assert.Equal(t, 2, svc.Client.(*cwatch.DevClient).CallCount())

	// create service for prod environment
	svc, err = cwatch.NewService("root", "key", "us-east-1", "Foo", "prod")
	assert.NoError(t, err)
	assert.NotNil(t, svc)
}
