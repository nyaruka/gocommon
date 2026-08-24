package dynamo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nyaruka/gocommon/spools"
)

// spooled is a DynamoDB write (a put or a delete) and the table it's destined for.
type spooled struct {
	table  string
	item   map[string]types.AttributeValue // item attrs for a put, key attrs for a delete
	delete bool
}

func (s *spooled) request() types.WriteRequest {
	if s.delete {
		return types.WriteRequest{DeleteRequest: &types.DeleteRequest{Key: s.item}}
	}
	return types.WriteRequest{PutRequest: &types.PutRequest{Item: s.item}}
}

func spooledFromRequest(table string, req types.WriteRequest) *spooled {
	if req.DeleteRequest != nil {
		return &spooled{table: table, item: req.DeleteRequest.Key, delete: true}
	}
	return &spooled{table: table, item: req.PutRequest.Item}
}

// spooledJSON is the serialized form of a spooled write. The delete marker is omitted for puts so files written by
// older versions (which only spooled puts) read back correctly.
type spooledJSON struct {
	Table  string          `json:"table"`
	Item   json.RawMessage `json:"item"`
	Delete bool            `json:"delete,omitempty"`
}

func marshalSpooled(s *spooled) ([]byte, error) {
	item, err := attributevalue.MarshalMapJSON(s.item)
	if err != nil {
		return nil, fmt.Errorf("error marshaling item: %w", err)
	}
	return json.Marshal(&spooledJSON{Table: s.table, Item: item, Delete: s.delete})
}

func unmarshalSpooled(data []byte) (*spooled, error) {
	sj := &spooledJSON{}
	if err := json.Unmarshal(data, sj); err != nil {
		return nil, err
	}
	item, err := attributevalue.UnmarshalMapJSON(sj.Item)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling item: %w", err)
	}
	return &spooled{table: sj.Table, item: item, delete: sj.Delete}, nil
}

// Spool writes DynamoDB writes (puts and deletes) to local files and periodically retries them against DynamoDB.
//
// Flushing is at-least-once so writes may be replayed after a crash - see [spools.Spool]. Note this means a replayed
// put can resurrect an item deleted since it was spooled, and a replayed delete can remove an item recreated since.
type Spool struct {
	client *dynamodb.Client
	spool  *spools.Spool[*spooled]
}

// NewSpool creates a new spool using the given directory and flush interval.
func NewSpool(client *dynamodb.Client, directory string, flushInterval time.Duration) *Spool {
	s := &Spool{client: client}
	s.spool = spools.New(directory, flushInterval, marshalSpooled, unmarshalSpooled, s.flushBatch)
	return s
}

// Start starts the spool's background flushing - see [spools.Spool.Start].
func (s *Spool) Start() error {
	return s.spool.Start()
}

// Stop stops the spool's background flushing - see [spools.Spool.Stop].
func (s *Spool) Stop() {
	s.spool.Stop()
}

// Add writes items to be put in the given table to a new spool file.
func (s *Spool) Add(table string, items []map[string]types.AttributeValue) error {
	batch := make([]*spooled, len(items))
	for i, item := range items {
		batch[i] = &spooled{table: table, item: item}
	}
	return s.spool.Add(batch)
}

// AddWrites writes put and delete requests destined for the given table to a new spool file.
func (s *Spool) AddWrites(table string, requests []types.WriteRequest) error {
	batch := make([]*spooled, len(requests))
	for i, req := range requests {
		batch[i] = spooledFromRequest(table, req)
	}
	return s.spool.Add(batch)
}

// Flush performs an immediate flush of all spooled files - see [spools.Spool.Flush].
func (s *Spool) Flush() error {
	return s.spool.Flush()
}

// Size returns the number of items currently spooled.
func (s *Spool) Size() int {
	return s.spool.Size()
}

// Delete removes the spool directory and all spooled files.
func (s *Spool) Delete() error {
	return s.spool.Delete()
}

func (s *Spool) flushBatch(ctx context.Context, batch []*spooled) ([]*spooled, error) {
	// group by table preserving order - though in practice a spool file is written from a single writer batch so all
	// of its writes are for the same table
	tables := make([]string, 0, 1)
	byTable := make(map[string][]types.WriteRequest, 1)
	for _, sp := range batch {
		if _, seen := byTable[sp.table]; !seen {
			tables = append(tables, sp.table)
		}
		byTable[sp.table] = append(byTable[sp.table], sp.request())
	}

	var failed []*spooled
	for _, table := range tables {
		unprocessed, err := batchWriteItem(ctx, s.client, table, byTable[table])
		if err != nil {
			return nil, err
		}
		for _, req := range unprocessed {
			failed = append(failed, spooledFromRequest(table, req))
		}
	}
	return failed, nil
}
