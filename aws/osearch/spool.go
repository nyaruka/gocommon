package osearch

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nyaruka/gocommon/uuids"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

type spooledFile struct {
	path   string
	count  int
	action Action
	index  string
}

var spooledFileRegex = regexp.MustCompile(`^[^@]+#(\d+)@(\w+)@(.+)\.jsonl$`) // <uuid>#<count>@<action>@<index>.jsonl

// Spool writes OpenSearch documents to local files and periodically retries indexing them.
type Spool struct {
	client        *opensearchapi.Client
	directory     string
	size          atomic.Int64
	flushInterval time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewSpool(client *opensearchapi.Client, directory string, flushInterval time.Duration) *Spool {
	ctx, cancel := context.WithCancel(context.Background())

	return &Spool{
		client:        client,
		directory:     directory,
		flushInterval: flushInterval,
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (s *Spool) Start() error {
	// ensure directory exists
	if err := os.MkdirAll(s.directory, 0755); err != nil {
		return fmt.Errorf("error creating spool directory %s: %w", s.directory, err)
	}

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
				if err := s.flush(); err != nil {
					slog.Error("error flushing spool", "error", err)
				}
			}
		}
	}()

	return nil
}

func (s *Spool) Stop() {
	s.cancel()

	s.wg.Wait()
}

func (s *Spool) Add(index string, action Action, items [][]byte) error {
	path := fmt.Sprintf("%s/%s#%d@%s@%s.jsonl", s.directory, uuids.NewV7(), len(items), action, index)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating spool file %s: %w", path, err)
	}
	defer f.Close()

	for _, item := range items {
		if _, err := f.Write(item); err != nil {
			return fmt.Errorf("error writing item to spool file %s: %w", path, err)
		}
		if _, err := f.Write([]byte("\n")); err != nil {
			return fmt.Errorf("error writing item to spool file %s: %w", path, err)
		}

		s.size.Add(1)
	}

	return nil
}

func (s *Spool) flush() error {
	ctx := context.TODO()

	files, err := s.enumerateFiles()
	if err != nil {
		return fmt.Errorf("error enumerating files to flush: %w", err)
	}

	for _, file := range files {
		items, err := s.readFile(file.path)
		if err != nil {
			return fmt.Errorf("error loading spool file %s: %w", file.path, err)
		}

		_, unprocessed, err := bulk(ctx, s.client, file.index, file.action, items)
		if err != nil {
			slog.Error("error flushing spooled opensearch batch", "error", err, "file", file.path)
			continue
		}

		if len(unprocessed) > 0 {
			// write unprocessed items back to a new spool file
			if err := s.Add(file.index, file.action, unprocessed); err != nil {
				return fmt.Errorf("error writing unprocessed items back to spool file %s: %w", file.path, err)
			}
		}

		if err := os.Remove(file.path); err != nil {
			return fmt.Errorf("error removing spool file %s: %w", file.path, err)
		}

		s.size.Add(-int64(file.count))
	}

	return nil
}

func (s *Spool) readFile(path string) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening spool file %s: %w", path, err)
	}
	defer f.Close()

	items := make([][]byte, 0)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB max line size
	for scanner.Scan() {
		items = append(items, bytes.Clone(scanner.Bytes()))
	}

	return items, nil
}

func (s *Spool) enumerateFiles() ([]spooledFile, error) {
	files := make([]spooledFile, 0)

	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return nil, fmt.Errorf("error listing spool directory %s: %w", s.directory, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			matches := spooledFileRegex.FindStringSubmatch(entry.Name())
			if len(matches) == 4 {
				path := fmt.Sprintf("%s/%s", s.directory, entry.Name())
				count, _ := strconv.Atoi(matches[1])
				files = append(files, spooledFile{path: path, count: count, action: Action(matches[2]), index: matches[3]})
			}
		}
	}
	return files, nil
}

func (s *Spool) Size() int {
	return int(s.size.Load())
}

func (s *Spool) Delete() error {
	return os.RemoveAll(s.directory)
}
