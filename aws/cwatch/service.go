package cwatch

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/nyaruka/gocommon/syncx"
)

type Service struct {
	Client    *cloudwatch.Client
	namespace string
	batcher   *syncx.Batcher[types.MetricDatum]
}

func NewService(accessKey, secretKey, region, namespace string, wg *sync.WaitGroup) (*Service, error) {
	opts := []func(*config.LoadOptions) error{config.WithRegion(region)}

	if accessKey != "" && secretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.StaticCredentialsProvider{Value: aws.Credentials{
			AccessKeyID: accessKey, SecretAccessKey: secretKey,
		}}))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), opts...)
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
