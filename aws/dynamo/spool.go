package dynamo

import (
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/nyaruka/gocommon/uuids"
)

type Spool[I any] struct {
	directory string
	size      atomic.Int64
}

func NewSpool[I any](directory string) *Spool[I] {
	return &Spool[I]{directory: directory}
}

func (s *Spool[I]) Start() error {
	return os.MkdirAll(s.directory, 0755)
}

func (s *Spool[I]) Write(items []*I) error {
	path := fmt.Sprintf("%s/%s.jsonl", s.directory, uuids.NewV7())
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating spool file %s: %w", path, err)
	}
	defer f.Close()

	for _, item := range items {
		d, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("error marshaling item to JSON: %w", err)
		}
		d = append(d, '\n')

		if _, err := f.Write(d); err != nil {
			return fmt.Errorf("error writing item to spool file %s: %w", path, err)
		}

		s.size.Add(1)
	}

	return nil
}

func (s *Spool[I]) Size() int {
	return int(s.size.Load())
}

func (s *Spool[I]) Delete() error {
	return os.RemoveAll(s.directory)
}
