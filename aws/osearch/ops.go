package osearch

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// Action is the bulk action type for OpenSearch documents.
type Action string

const (
	// ActionIndex is the bulk action for indexing documents. If a document with the same ID already exists, it will be
	// replaced. Use this for regular search indexes.
	ActionIndex Action = "index"

	// ActionCreate is the bulk action for creating documents. If a document with the same ID already exists, the
	// operation will fail. Use this for time-series indexes where documents are always new.
	ActionCreate Action = "create"
)

// BulkIndex sends a batch of JSON documents to OpenSearch using the index action.
func BulkIndex(ctx context.Context, client *opensearchapi.Client, index string, items [][]byte) (int, [][]byte, error) {
	return bulk(ctx, client, index, ActionIndex, items)
}

// BulkCreate sends a batch of JSON documents to OpenSearch using the create action.
func BulkCreate(ctx context.Context, client *opensearchapi.Client, index string, items [][]byte) (int, [][]byte, error) {
	return bulk(ctx, client, index, ActionCreate, items)
}

func bulk(ctx context.Context, client *opensearchapi.Client, index string, action Action, items [][]byte) (int, [][]byte, error) {
	if len(items) == 0 {
		return 0, nil, nil
	}

	actionLine := []byte(`{"` + string(action) + `":{}}`)

	var buf bytes.Buffer
	for _, item := range items {
		buf.Write(actionLine)
		buf.WriteByte('\n')
		buf.Write(item)
		buf.WriteByte('\n')
	}

	resp, err := client.Bulk(ctx, opensearchapi.BulkReq{Index: index, Body: &buf})
	if err != nil {
		return 0, nil, fmt.Errorf("error sending bulk request to opensearch index %s: %w", index, err)
	}

	if !resp.Errors {
		return len(items), nil, nil
	}

	numWritten := 0
	var retryable [][]byte

	for i, itemMap := range resp.Items {
		for _, item := range itemMap {
			if item.Status >= 200 && item.Status < 300 {
				numWritten++
			} else if item.Status == 429 || item.Status >= 500 {
				retryable = append(retryable, items[i])
			} else {
				errType, errReason := "", ""
				if item.Error != nil {
					errType = item.Error.Type
					errReason = item.Error.Reason
				}
				slog.Error("permanent opensearch bulk index failure", "index", index, "status", item.Status, "error_type", errType, "error_reason", errReason)
			}
		}
	}

	return numWritten, retryable, nil
}
