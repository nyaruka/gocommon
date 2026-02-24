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
	path  string
	count int
	index string
}

var spooledFileRegex = regexp.MustCompile(`^[^@]+#(\d+)@(.+)\.jsonl$`) // <uuid>#<count>@<index>.jsonl

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

// Add writes documents to spool files, grouping by index. Each index gets its own spool file.
func (s *Spool) Add(docs []*Document) error {
	byIndex := make(map[string][][]byte)
	for _, doc := range docs {
		byIndex[doc.Index] = append(byIndex[doc.Index], doc.Body)
	}
	for index, items := range byIndex {
		if err := s.writeFile(index, items); err != nil {
			return err
		}
	}
	return nil
}

func (s *Spool) writeFile(index string, items [][]byte) error {
	path := fmt.Sprintf("%s/%s#%d@%s.jsonl", s.directory, uuids.NewV7(), len(items), index)
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

		docs := make([]*Document, len(items))
		for i, item := range items {
			docs[i] = &Document{Index: file.index, Body: item}
		}

		_, unprocessed, err := BulkIndex(ctx, s.client, docs)
		if err != nil {
			slog.Error("error flushing spooled opensearch batch", "error", err, "file", file.path)
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
			if len(matches) == 3 {
				path := fmt.Sprintf("%s/%s", s.directory, entry.Name())
				count, _ := strconv.Atoi(matches[1])
				files = append(files, spooledFile{path: path, count: count, index: matches[2]})
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
