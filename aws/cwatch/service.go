package cwatch

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	awsx "github.com/nyaruka/gocommon/aws"
	"github.com/nyaruka/gocommon/syncx"
)

type Service struct {
	Client     Client
	namespace  string
	deployment types.Dimension
	batcher    *syncx.Batcher[types.MetricDatum]
}

// NewService creates a new Cloudwatch service with the given credentials and configuration. If deployment is "dev" then
// then instead of a real Cloudwatch client, the service will get a mocked version that just logs metrics.
func NewService(accessKey, secretKey, region, namespace, deployment string) (*Service, error) {
	var client Client

	if deployment == "dev" {
		client = &DevClient{}
	} else {
		cfg, err := awsx.NewConfig(accessKey, secretKey, region)
		if err != nil {
			return nil, err
		}
		client = cloudwatch.NewFromConfig(cfg)
	}

	return &Service{
		Client:     client,
		namespace:  namespace,
		deployment: types.Dimension{Name: aws.String("Deployment"), Value: aws.String(deployment)},
	}, nil
}

func (s *Service) StartQueue(wg *sync.WaitGroup, maxAge time.Duration) {
	if s.batcher != nil {
		panic("queue already started")
	}
	s.batcher = syncx.NewBatcher(s.processBatch, 100, maxAge, 1000, wg)
	s.batcher.Start()
}

func (s *Service) StopQueue() {
	if s.batcher == nil {
		panic("queue wasn't started")
	}
	s.batcher.Stop()
}

func (s *Service) Queue(d types.MetricDatum) {
	s.batcher.Queue(d)
}

func (s *Service) Prepare(data []types.MetricDatum) *cloudwatch.PutMetricDataInput {
	// add deployment as the first dimension to all metrics
	for i := range data {
		data[i].Dimensions = append([]types.Dimension{s.deployment}, data[i].Dimensions...)
	}

	return &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(s.namespace),
		MetricData: data,
	}
}

func (s *Service) processBatch(batch []types.MetricDatum) {
	_, err := s.Client.PutMetricData(context.TODO(), s.Prepare(batch))
	if err != nil {
		slog.Error("error sending metric data batch", "error", err, "count", len(batch))
	}
}
