package analytics

// MockBackend is a backend which records values for testing
type MockBackend struct {
	Gauges map[string][]float64
}

// NewMock creates a new mock backend
func NewMock() *MockBackend {
	return &MockBackend{Gauges: make(map[string][]float64)}
}

func (b *MockBackend) Name() string {
	return "mock"
}

func (b *MockBackend) Start() error {
	return nil
}

func (b *MockBackend) Gauge(name string, value float64) {
	b.Gauges[name] = append(b.Gauges[name], value)
}

func (b *MockBackend) Stop() error {
	return nil
}

var _ Backend = (*MockBackend)(nil)
