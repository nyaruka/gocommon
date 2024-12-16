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
	deployment string

	batcher   *syncx.Batcher[types.MetricDatum]
	batcherWG *sync.WaitGroup
}

// NewService creates a new Cloudwatch service with the given credentials and configuration. Some behaviours depend on
// the given deployment value:
//   - "test": metrics just logged, Queue(..) sends synchronously
//   - "dev": metrics just logged, Queue(..) adds to batcher
//   - "*": metrics sent to Cloudwatch, Queue(..) adds to batcher
func NewService(accessKey, secretKey, region, namespace, deployment string) (*Service, error) {
	var client Client

	if deployment == "dev" || deployment == "test" {
		client = &DevClient{}
	} else {
		cfg, err := awsx.NewConfig(accessKey, secretKey, region)
		if err != nil {
			return nil, err
		}
		client = cloudwatch.NewFromConfig(cfg)
	}

	return &Service{Client: client, namespace: namespace, deployment: deployment}, nil
}

func (s *Service) StartQueue(maxAge time.Duration) {
	if s.batcher != nil {
		panic("queue already started")
	}

	s.batcherWG = &sync.WaitGroup{}
	s.batcher = syncx.NewBatcher(s.processBatch, 100, maxAge, 1000, s.batcherWG)
	s.batcher.Start()
}

func (s *Service) StopQueue() {
	if s.batcher == nil {
		panic("queue wasn't started")
	}
	s.batcher.Stop()
	s.batcherWG.Wait()
}

func (s *Service) Queue(data ...types.MetricDatum) {
	if s.deployment == "test" {
		s.Send(context.TODO(), data...)
	} else {
		for _, d := range data {
			s.batcher.Queue(d)
		}
	}
}

func (s *Service) Send(ctx context.Context, data ...types.MetricDatum) error {
	_, err := s.Client.PutMetricData(ctx, s.prepare(data))
	return err
}

func (s *Service) prepare(data []types.MetricDatum) *cloudwatch.PutMetricDataInput {
	// add deployment as the first dimension to all metrics
	for i := range data {
		data[i].Dimensions = append([]types.Dimension{{Name: aws.String("Deployment"), Value: aws.String(s.deployment)}}, data[i].Dimensions...)
	}

	return &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(s.namespace),
		MetricData: data,
	}
}

func (s *Service) processBatch(batch []types.MetricDatum) {
	if err := s.Send(context.TODO(), batch...); err != nil {
		slog.Error("error sending metric data batch", "error", err, "count", len(batch))
	}
}
