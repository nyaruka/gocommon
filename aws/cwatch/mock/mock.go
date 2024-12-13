package mock

import (
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/nyaruka/gocommon/aws/cwatch"
)

type MockCWService struct {
	namespace  string
	deployment types.Dimension
	Batcher    []types.MetricDatum

	Stopped bool
}

func NewMockCWService(accessKey, secretKey, region, namespace, deployment string) (*MockCWService, error) {
	mockCW := MockCWService{
		namespace:  namespace,
		deployment: types.Dimension{Name: aws.String("Deployment"), Value: aws.String(deployment)},
		Batcher:    nil,
	}

	return &mockCW, nil
}

func (s *MockCWService) Queue(d types.MetricDatum) {
	if s.Stopped {
		return
	}
	s.Batcher = append(s.Batcher, d)
}

func (s *MockCWService) StartQueue(wg *sync.WaitGroup) {
	s.Batcher = []types.MetricDatum{}
	s.Stopped = false
}

func (s *MockCWService) StopQueue() {
	s.Stopped = true
}

var _ cwatch.CWService = (*MockCWService)(nil)
