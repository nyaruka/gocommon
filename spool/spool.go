// Package spool provides a generic file-backed spool for items which couldn't be written to their primary store and
// need to be retried later.
package spool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nyaruka/gocommon/uuids"
)

// FlushFunc attempts the writing of a batch of previously spooled items to their primary store. It returns any items
// which couldn't be written and should be respooled, or an error if the batch as a whole couldn't be attempted, in
// which case it will be retried later in its entirety.
type FlushFunc[T any] func(ctx context.Context, batch []T) (failed []T, err error)

// MarshalJSON is a marshal function for spools whose items serialize as plain JSON.
func MarshalJSON[T any](item T) ([]byte, error) { return json.Marshal(item) }

// UnmarshalJSON is an unmarshal function for spools whose items serialize as plain JSON.
func UnmarshalJSON[T any](data []byte) (T, error) {
	var item T
	err := json.Unmarshal(data, &item)
	return item, err
}

type spooledFile struct {
	path  string
	count int
}

var spooledFileRegex = regexp.MustCompile(`^[^#]+#(\d+)\.jsonl$`) // <uuid>#<count>.jsonl

const maxLineSize = 1024 * 1024 // 1MB

// Spool writes batches of items to local JSONL files and periodically retries writing them to their primary store
// using a flush function.
//
// Flushing is at-least-once: a crash between a successful flush and removal of the flushed file means that the file's
// entire batch is replayed on restart. Writes performed by the flush function must therefore be idempotent or
// deduplicated downstream.
type Spool[T any] struct {
	directory     string
	flushInterval time.Duration
	marshal       func(T) ([]byte, error)
	unmarshal     func([]byte) (T, error)
	flush         FlushFunc[T]

	size    atomic.Int64
	flushMu sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new spool which stores items in the given directory, marshaling and unmarshaling individual items
// with the given functions, and retrying batches with the given flush function every flushInterval.
func New[T any](directory string, flushInterval time.Duration, marshal func(T) ([]byte, error), unmarshal func([]byte) (T, error), flush FlushFunc[T]) *Spool[T] {
	ctx, cancel := context.WithCancel(context.Background())

	return &Spool[T]{
		directory:     directory,
		flushInterval: flushInterval,
		marshal:       marshal,
		unmarshal:     unmarshal,
		flush:         flush,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start ensures the spool directory exists and is writable, restores the size count from any existing spool files,
// and starts the background flush loop.
func (s *Spool[T]) Start() error {
	// ensure directory exists
	if err := os.MkdirAll(s.directory, 0755); err != nil {
		return fmt.Errorf("error creating spool directory %s: %w", s.directory, err)
	}

	// MkdirAll succeeds if directory already exists even if it's not writable, so probe actual writability
	probe, err := os.CreateTemp(s.directory, ".probe-*")
	if err != nil {
		return fmt.Errorf("spool directory %s is not writable: %w", s.directory, err)
	}
	probe.Close()
	os.Remove(probe.Name())

	// enumerate existing files to get current size
	files, err := s.enumerateFiles()
	if err != nil {
		return err
	}
	total := 0
	for _, file := range files {
		total += file.count
	}
	s.size.Store(int64(total))

	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(s.flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				if err := s.flushAll(); err != nil {
					slog.Error("error flushing spool", "error", err)
				}
			}
		}
	}()

	return nil
}

// Stop stops the background flush loop, waiting for any in progress flush to complete.
func (s *Spool[T]) Stop() {
	s.cancel()

	s.wg.Wait()
}

// Add writes items to a new spool file. The file is written with a temporary name and renamed into place so that a
// partially written file is never eligible for flushing.
func (s *Spool[T]) Add(items []T) error {
	path := fmt.Sprintf("%s/%s#%d.jsonl", s.directory, uuids.NewV7(), len(items))
	temp := path + ".tmp"

	if err := s.writeFile(temp, items); err != nil {
		os.Remove(temp)
		return err
	}

	if err := os.Rename(temp, path); err != nil {
		os.Remove(temp)
		return fmt.Errorf("error renaming spool file %s: %w", temp, err)
	}

	s.size.Add(int64(len(items)))

	return nil
}

// Flush performs an immediate flush of all spooled files. Flushing normally happens on the flush interval so this is
// mostly useful in tests.
func (s *Spool[T]) Flush() error {
	return s.flushAll()
}

// Size returns the number of items currently spooled.
func (s *Spool[T]) Size() int {
	return int(s.size.Load())
}

// Delete removes the spool directory and all spooled files.
func (s *Spool[T]) Delete() error {
	return os.RemoveAll(s.directory)
}

func (s *Spool[T]) writeFile(path string, items []T) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating spool file %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, item := range items {
		marshaled, err := s.marshal(item)
		if err != nil {
			return fmt.Errorf("error marshaling item for spool file %s: %w", path, err)
		}
		if _, err := w.Write(marshaled); err != nil {
			return fmt.Errorf("error writing item to spool file %s: %w", path, err)
		}
		if err := w.WriteByte('\n'); err != nil {
			return fmt.Errorf("error writing item to spool file %s: %w", path, err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("error writing spool file %s: %w", path, err)
	}

	return nil
}

func (s *Spool[T]) flushAll() error {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	ctx := s.ctx

	files, err := s.enumerateFiles()
	if err != nil {
		return fmt.Errorf("error enumerating files to flush: %w", err)
	}

	for _, file := range files {
		items, err := s.readFile(file.path)
		if err != nil {
			// don't let one unreadable file prevent others from being flushed
			slog.Error("error reading spool file", "error", err, "file", file.path)
			continue
		}

		failed, err := s.flush(ctx, items)
		if err != nil {
			slog.Error("error flushing spooled batch", "error", err, "file", file.path)
			continue
		}

		if len(failed) > 0 {
			// write failed items back to a new spool file
			if err := s.Add(failed); err != nil {
				return fmt.Errorf("error respooling failed items from spool file %s: %w", file.path, err)
			}
		}

		if err := os.Remove(file.path); err != nil {
			return fmt.Errorf("error removing spool file %s: %w", file.path, err)
		}
	}

	// refresh size from disk to pick up any manual file changes
	files, err = s.enumerateFiles()
	if err != nil {
		return fmt.Errorf("error enumerating files after flush: %w", err)
	}
	total := 0
	for _, file := range files {
		total += file.count
	}
	s.size.Store(int64(total))

	return nil
}

func (s *Spool[T]) readFile(path string) ([]T, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening spool file %s: %w", path, err)
	}
	defer f.Close()

	var items []T

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, maxLineSize), maxLineSize)
	for scanner.Scan() {
		item, err := s.unmarshal(scanner.Bytes())
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling item from spool file %s: %w", path, err)
		}
		items = append(items, item)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading spool file %s: %w", path, err)
	}

	return items, nil
}

func (s *Spool[T]) enumerateFiles() ([]spooledFile, error) {
	files := make([]spooledFile, 0)

	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return nil, fmt.Errorf("error listing spool directory %s: %w", s.directory, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			matches := spooledFileRegex.FindStringSubmatch(entry.Name())
			if len(matches) == 2 {
				path := fmt.Sprintf("%s/%s", s.directory, entry.Name())
				count, _ := strconv.Atoi(matches[1])
				files = append(files, spooledFile{path: path, count: count})
			}
		}
	}
	return files, nil
}
