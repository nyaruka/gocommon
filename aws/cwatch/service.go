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
	Client     *cloudwatch.Client
	namespace  string
	deployment types.Dimension
	batcher    *syncx.Batcher[types.MetricDatum]
}

// NewService creates a new Cloudwatch service with the given credentials and configuration
func NewService(accessKey, secretKey, region, namespace, deployment string, wg *sync.WaitGroup) (*Service, error) {
	cfg, err := awsx.NewConfig(accessKey, secretKey, region)
	if err != nil {
		return nil, err
	}

	s := &Service{
		Client:     cloudwatch.NewFromConfig(cfg),
		namespace:  namespace,
		deployment: types.Dimension{Name: aws.String("Deployment"), Value: aws.String(deployment)},
	}
	s.batcher = syncx.NewBatcher(s.processBatch, 100, time.Second*3, 1000, wg)

	return s, nil
}

func (s *Service) Start() {
	s.batcher.Start()
}

func (s *Service) Stop() {
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
		slog.Error("error sending metrics to cloudwatch", "error", err, "count", len(batch))
	}
}
