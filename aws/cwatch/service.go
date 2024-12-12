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
	Client    *cloudwatch.Client
	namespace string
	batcher   *syncx.Batcher[types.MetricDatum]
}

// NewService creates a new Cloudwatch service with the given credentials and configuration
func NewService(accessKey, secretKey, region, namespace string, wg *sync.WaitGroup) (*Service, error) {
	cfg, err := awsx.NewConfig(accessKey, secretKey, region)
	if err != nil {
		return nil, err
	}

	client := cloudwatch.NewFromConfig(cfg)
	s := &Service{Client: client, namespace: namespace}
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

func (s *Service) processBatch(batch []types.MetricDatum) {
	_, err := s.Client.PutMetricData(context.TODO(), &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(s.namespace),
		MetricData: batch,
	})
	if err != nil {
		slog.Error("error sending metrics to cloudwatch", "error", err, "count", len(batch))
	}
}
