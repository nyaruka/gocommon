package cwatch_test

import (
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/nyaruka/gocommon/aws/cwatch"
	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	wg := &sync.WaitGroup{}

	svc, err := cwatch.NewService("root", "key", "us-east-1", "Foo", "testing", wg)
	assert.NoError(t, err)

	assert.Equal(t, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("Foo"),
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String("NumGoats"),
				Dimensions: []types.Dimension{
					{Name: aws.String("Deployment"), Value: aws.String("testing")},
				},
				Value: aws.Float64(10),
			},
			{
				MetricName: aws.String("NumSheep"),
				Dimensions: []types.Dimension{
					{Name: aws.String("Deployment"), Value: aws.String("testing")},
					{Name: aws.String("Host"), Value: aws.String("foo1")},
				},
				Value: aws.Float64(20),
			},
		},
	}, svc.Prepare([]types.MetricDatum{
		{MetricName: aws.String("NumGoats"), Value: aws.Float64(10)},
		{MetricName: aws.String("NumSheep"), Dimensions: []types.Dimension{{Name: aws.String("Host"), Value: aws.String("foo1")}}, Value: aws.Float64(20)},
	}))

	svc.Start()

	svc.Stop()

	wg.Wait()
}
