package dynamo

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/uuids"
)

type spooledFile struct {
	path  string
	count int
	table string
}

var spooledFileRegex = regexp.MustCompile(`^[^@]+#(\d+)@(\w+)\.jsonl$`) // <uuid>#<count>@<table>.jsonl

// Spool writes DynamoDB items to local files and periodically retries putting them in DynamoDB.
type Spool struct {
	client        *dynamodb.Client
	directory     string
	size          atomic.Int64
	flushInterval time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewSpool(client *dynamodb.Client, directory string, flushInterval time.Duration) *Spool {
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

func (s *Spool) Add(table string, items []map[string]types.AttributeValue) error {
	path := fmt.Sprintf("%s/%s#%d@%s.jsonl", s.directory, uuids.NewV7(), len(items), table)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating spool file %s: %w", path, err)
	}
	defer f.Close()

	for _, item := range items {
		marshaled, err := attributevalue.MarshalMapJSON(item)
		if err != nil {
			return fmt.Errorf("error marshaling item to JSON for spool file %s: %w", path, err)
		}

		marshaled = append(marshaled, []byte("\n")...)

		if _, err := f.Write(marshaled); err != nil {
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

		unprocessed, err := batchPutItem(ctx, s.client, file.table, items)
		if err != nil {
			slog.Error("error flushing spooled dynamo batch", "error", err, "file", file.path)
			continue
		}

		if len(unprocessed) > 0 {
			// write unprocessed items back to a new spool file
			if err := s.Add(file.table, unprocessed); err != nil {
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

func (s *Spool) readFile(path string) ([]map[string]types.AttributeValue, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening spool file %s: %w", path, err)
	}
	defer f.Close()

	items := make([]map[string]types.AttributeValue, 0, 25)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		item, err := attributevalue.UnmarshalMapJSON(scanner.Bytes())
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling item from spool file %s: %w", path, err)
		}
		items = append(items, item)
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
				files = append(files, spooledFile{path: path, count: count, table: matches[2]})
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
