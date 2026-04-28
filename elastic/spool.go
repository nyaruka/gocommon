package elastic

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

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/nyaruka/gocommon/uuids"
)

type spooledFile struct {
	path  string
	count int
}

var spooledFileRegex = regexp.MustCompile(`^[^#]+#(\d+)\.jsonl$`) // <uuid>#<count>.jsonl

// Spool writes Elasticsearch documents to local files and periodically retries indexing them.
type Spool struct {
	client        *elasticsearch.TypedClient
	directory     string
	size          atomic.Int64
	flushInterval time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewSpool(client *elasticsearch.TypedClient, directory string, flushInterval time.Duration) *Spool {
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

// Add writes documents to a spool file.
func (s *Spool) Add(docs []*Document) error {
	path := fmt.Sprintf("%s/%s#%d.jsonl", s.directory, uuids.NewV7(), len(docs))
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating spool file %s: %w", path, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, doc := range docs {
		if err := enc.Encode(doc); err != nil {
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
		docs, err := s.readFile(file.path)
		if err != nil {
			return fmt.Errorf("error loading spool file %s: %w", file.path, err)
		}

		_, unprocessed, err := Bulk(ctx, s.client, docs)
		if err != nil {
			slog.Error("error flushing spooled elasticsearch batch", "error", err, "file", file.path)
			continue
		}

		if len(unprocessed) > 0 {
			if err := s.Add(unprocessed); err != nil {
				return fmt.Errorf("error writing unprocessed items back to spool file %s: %w", file.path, err)
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

func (s *Spool) readFile(path string) ([]*Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening spool file %s: %w", path, err)
	}
	defer f.Close()

	var docs []*Document

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB max line size
	for scanner.Scan() {
		var doc Document
		if err := json.Unmarshal(scanner.Bytes(), &doc); err != nil {
			return nil, fmt.Errorf("error unmarshalling spool line in %s: %w", path, err)
		}
		docs = append(docs, &doc)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading spool file %s: %w", path, err)
	}

	return docs, nil
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
			if len(matches) == 2 {
				path := fmt.Sprintf("%s/%s", s.directory, entry.Name())
				count, _ := strconv.Atoi(matches[1])
				files = append(files, spooledFile{path: path, count: count})
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
