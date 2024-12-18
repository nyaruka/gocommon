package cwatch_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/nyaruka/gocommon/aws/cwatch"
	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	// create service for dev environment
	svc, err := cwatch.NewService("root", "key", "us-east-1", "Foo", "dev")
	assert.NoError(t, err)

	err = svc.Send(context.Background(), types.MetricDatum{MetricName: aws.String("NumSheep"), Dimensions: []types.Dimension{{Name: aws.String("Host"), Value: aws.String("foo1")}}, Value: aws.Float64(20)})
	assert.NoError(t, err)
	assert.Equal(t, 1, svc.Client.(*cwatch.DevClient).CallCount())

	// create service for prod environment
	svc, err = cwatch.NewService("root", "key", "us-east-1", "Foo", "prod")
	assert.NoError(t, err)
	assert.NotNil(t, svc)
}
