package cwatch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	awsx "github.com/nyaruka/gocommon/aws"
)

type Service struct {
	Client     Client
	namespace  string
	deployment string
}

// NewService creates a new Cloudwatch service with the given credentials and configuration. If deployment is given as
// "dev" or "test", then metrics are logged and not sent to Cloudwatch.
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
