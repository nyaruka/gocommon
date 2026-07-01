package cwatch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type Service struct {
	Client     Client
	namespace  string
	deployment string
}

// NewService creates a new Cloudwatch service, resolving credentials and region from the standard AWS SDK default
// chain. If deployment is given as "dev" or "test", then metrics are logged and not sent to Cloudwatch.
func NewService(ctx context.Context, namespace, deployment string) (*Service, error) {
	var client Client

	if deployment == "dev" || deployment == "test" {
		client = &DevClient{}
	} else {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, err
		}
		client = cloudwatch.NewFromConfig(cfg)
	}

	return &Service{Client: client, namespace: namespace, deployment: deployment}, nil
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
