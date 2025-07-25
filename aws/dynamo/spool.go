package dynamo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nyaruka/gocommon/uuids"
)

var spooledFileRegex = regexp.MustCompile(`^[^@]+@(\d+)\.jsonl$`) // <uuid>@<count>.jsonl

type Spool[I any] struct {
	directory string
	size      atomic.Int64
	wg        *sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc
}

func NewSpool[I any](ctx context.Context, directory string, wg *sync.WaitGroup) *Spool[I] {
	ctx, cancel := context.WithCancel(ctx)
	return &Spool[I]{
		directory: directory,
		wg:        wg,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (s *Spool[I]) Start() error {
	// ensure directory exists
	if err := os.MkdirAll(s.directory, 0755); err != nil {
		return fmt.Errorf("error creating spool directory %s: %w", s.directory, err)
	}

	// enumerate existing files to get current size
	_, total, err := s.enumerateFiles()
	if err != nil {
		return err
	}
	s.size.Store(int64(total))

	// start flush goroutine
	s.wg.Add(1)
	go s.flushLoop()

	return nil
}

func (s *Spool[I]) flushLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			// perform one final flush before stopping
			s.flush()
			return
		case <-ticker.C:
			if err := s.flush(); err != nil {
				// TODO: handle flush errors (maybe log them)
				_ = err
			}
		}
	}
}

func (s *Spool[I]) Stop() {
	s.cancel()
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

func (s *Spool[I]) flush() error {
	names, _, err := s.enumerateFiles()
	if err != nil {
		return fmt.Errorf("error enumerating files to flush: %w", err)
	}

	for _, name := range names {
		items, err := s.readFile(name)
		if err != nil {
			return fmt.Errorf("error loading spool file %s: %w", name, err)
		}

		// TODO write items to the table. If items are written without error, remove the file. If there are unprocessed
		// items, write them back to the spool file. If there's an error, leave file as is for retry.
		fmt.Println(items)

		if err := os.Remove(fmt.Sprintf("%s/%s", s.directory, name)); err != nil {
			return fmt.Errorf("error removing spool file %s: %w", name, err)
		}
	}

	return nil
}

func (s *Spool[I]) readFile(name string) ([]*I, error) {
	path := fmt.Sprintf("%s/%s", s.directory, name)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening spool file %s: %w", path, err)
	}
	defer f.Close()

	var items []*I
	decoder := json.NewDecoder(f)
	for {
		var item I
		if err := decoder.Decode(&item); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error decoding item from file %s: %w", path, err)
		}
		items = append(items, &item)
	}
	return items, nil
}

func (s *Spool[I]) enumerateFiles() ([]string, int, error) {
	var names []string
	var total int

	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return nil, 0, fmt.Errorf("error listing spool directory %s: %w", s.directory, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			matches := spooledFileRegex.FindStringSubmatch(entry.Name())
			if len(matches) == 2 {
				count, _ := strconv.Atoi(matches[1])
				if count != 0 {
					names = append(names, entry.Name())
					total += count
				}
			}
		}
	}
	return names, total, nil
}

func (s *Spool[I]) Size() int {
	return int(s.size.Load())
}

func (s *Spool[I]) Delete() error {
	return os.RemoveAll(s.directory)
}
