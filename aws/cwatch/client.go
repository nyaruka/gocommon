package cwatch

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/smithy-go/middleware"
)

// Client is the interface for a Cloudwatch client that can only send metrics
type Client interface {
	PutMetricData(ctx context.Context, params *cloudwatch.PutMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error)
}

type DevClient struct {
	callCount atomic.Int32
}

func (c *DevClient) PutMetricData(ctx context.Context, params *cloudwatch.PutMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error) {
	// log each metric being "sent"
	for _, md := range params.MetricData {
		log := slog.With("namespace", aws.ToString(params.Namespace))

		for _, dim := range md.Dimensions {
			log = log.With(strings.ToLower(aws.ToString(dim.Name)), aws.ToString(dim.Value))
		}
		log.With("metric", aws.ToString(md.MetricName), "value", aws.ToFloat64(md.Value), "unit", md.Unit).Info("put metric data")
	}

	c.callCount.Add(1)

	return &cloudwatch.PutMetricDataOutput{ResultMetadata: middleware.Metadata{}}, nil
}

func (c *DevClient) CallCount() int {
	return int(c.callCount.Load())
}
