package dynamo

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/nyaruka/gocommon/uuids"
)

var spooledFileRegex = regexp.MustCompile(`^[^@]+@(\d+)\.jsonl$`) // <uuid>@<count>.jsonl

type Spool[I any] struct {
	directory string
	size      atomic.Int64
	wg        *sync.WaitGroup
}

func NewSpool[I any](directory string, wg *sync.WaitGroup) *Spool[I] {
	return &Spool[I]{directory: directory, wg: wg}
}

func (s *Spool[I]) Start() error {
	// ensure directory exists
	if err := os.MkdirAll(s.directory, 0755); err != nil {
		return fmt.Errorf("error creating spool directory %s: %w", s.directory, err)
	}

	// enumerate existing files to get current size
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return fmt.Errorf("error listing spool directory %s: %w", s.directory, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			matches := spooledFileRegex.FindStringSubmatch(entry.Name())
			if len(matches) == 2 {
				count, _ := strconv.Atoi(matches[1])
				s.size.Add(int64(count))
			}
		}
	}

	return nil
}

func (s *Spool[I]) Stop() {
	// TODO
}

func (s *Spool[I]) Write(items []*I) error {
	path := fmt.Sprintf("%s/%s@%d.jsonl", s.directory, uuids.NewV7(), len(items))
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
