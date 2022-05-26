package analytics

import (
	"sync"
	"time"

	"github.com/nyaruka/librato"
)

// LibratoBackend is a backend which sends analytics to Librato
type LibratoBackend struct {
	collector librato.Collector
}

func NewLibrato(username, token, source string, timeout time.Duration, waitGroup *sync.WaitGroup) *LibratoBackend {
	return &LibratoBackend{librato.NewCollector(username, token, source, timeout, waitGroup)}
}

func (b *LibratoBackend) Name() string {
	return "librato"
}

func (b *LibratoBackend) Start() error {
	b.collector.Start()
	return nil
}

func (b *LibratoBackend) Gauge(name string, value float64) {
	b.collector.Gauge(name, value)
}

func (b *LibratoBackend) Stop() error {
	b.collector.Stop()
	return nil
}

var _ Backend = (*LibratoBackend)(nil)
