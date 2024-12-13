package cwatch_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/nyaruka/gocommon/aws/cwatch"
	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	svc, err := cwatch.NewService("root", "key", "us-east-1", "Foo", "dev")
	assert.NoError(t, err)

	assert.Equal(t, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("Foo"),
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String("NumGoats"),
				Dimensions: []types.Dimension{
					{Name: aws.String("Deployment"), Value: aws.String("dev")},
				},
				Value: aws.Float64(10),
			},
			{
				MetricName: aws.String("NumSheep"),
				Dimensions: []types.Dimension{
					{Name: aws.String("Deployment"), Value: aws.String("dev")},
					{Name: aws.String("Host"), Value: aws.String("foo1")},
				},
				Value: aws.Float64(20),
			},
		},
	}, svc.Prepare([]types.MetricDatum{
		{MetricName: aws.String("NumGoats"), Value: aws.Float64(10)},
		{MetricName: aws.String("NumSheep"), Dimensions: []types.Dimension{{Name: aws.String("Host"), Value: aws.String("foo1")}}, Value: aws.Float64(20)},
	}))

	wg := &sync.WaitGroup{}
	svc.StartQueue(wg, time.Millisecond*100)

	// test writing metrics directly via the client
	_, err = svc.Client.PutMetricData(context.Background(), svc.Prepare([]types.MetricDatum{
		{MetricName: aws.String("NumGoats"), Value: aws.Float64(10), Unit: types.StandardUnitCount},
		{MetricName: aws.String("NumSheep"), Dimensions: []types.Dimension{{Name: aws.String("Host"), Value: aws.String("foo1")}}, Value: aws.Float64(20), Unit: types.StandardUnitCount},
	}))
	assert.NoError(t, err)

	// test queuing metrics to be sent by batching process
	svc.Queue(types.MetricDatum{MetricName: aws.String("SleepTime"), Value: aws.Float64(30), Unit: types.StandardUnitSeconds})

	svc.StopQueue()
	wg.Wait()

	// check the queued metric was sent
	assert.Equal(t, 2, svc.Client.(*cwatch.DevClient).CallCount())
}
