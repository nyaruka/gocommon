package analytics

import (
	"sync"
	"time"
)

// Collector is anything which can collect analytics
type Collector interface {
	Start()
	Gauge(name string, value float64)
	Stop()
}

var singleton Collector

// Librato configures a librato collector for analytics
func Librato(waitGroup *sync.WaitGroup, username string, token string, source string, timeout time.Duration) {
	singleton = NewLibrato(libratoEndpoint, username, token, source, timeout, waitGroup)
}

// Start starts the analytics collector
func Start() {
	singleton.Start()
}

// Gauge records a new gauge value
func Gauge(name string, value float64) {
	singleton.Gauge(name, value)
}

// Stop stops the analytics collector
func Stop() {
	singleton.Stop()
}
