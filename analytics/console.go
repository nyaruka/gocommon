package analytics

import (
	"fmt"
	"io"
)

// ConsoleBackend is a backend which prints to the console or any output stream
type ConsoleBackend struct {
	out io.Writer
}

// NewConsole creates a new console backend
func NewConsole(out io.Writer) *ConsoleBackend {
	return &ConsoleBackend{out}
}

func (b *ConsoleBackend) Name() string {
	return "console"
}

func (b *ConsoleBackend) Start() error {
	return nil
}

func (b *ConsoleBackend) Gauge(name string, value float64) {
	fmt.Fprintf(b.out, "[analytics] gauge=%s value=%.2f\n", name, value)
}

func (b *ConsoleBackend) Stop() error {
	return nil
}

var _ Backend = (*ConsoleBackend)(nil)
