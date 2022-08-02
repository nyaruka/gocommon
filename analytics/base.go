package analytics

import "github.com/pkg/errors"

// Backend is the interface for backends
type Backend interface {
	Name() string
	Start() error
	Gauge(string, float64)
	Stop() error
}

var backends []Backend

// RegisterBackend registers a new backend
func RegisterBackend(b Backend) {
	backends = append(backends, b)
}

// Start starts all backends
func Start() error {
	for _, b := range backends {
		if err := b.Start(); err != nil {
			return errors.Wrapf(err, "error starting %s analytics backend", b.Name())
		}
	}
	return nil
}

// Gauge records a gauge value on all backends
func Gauge(name string, value float64) {
	for _, b := range backends {
		b.Gauge(name, value)
	}
}

// Stop stops all backends
func Stop() error {
	for _, b := range backends {
		if err := b.Stop(); err != nil {
			return errors.Wrapf(err, "error stopping %s analytics backend", b.Name())
		}
	}
	return nil
}
