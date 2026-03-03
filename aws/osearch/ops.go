package osearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// Document is a document to be indexed in OpenSearch.
type Document struct {
	Index   string          `json:"index"`
	ID      string          `json:"id"`
	Routing string          `json:"routing"`
	Version int64           `json:"version,omitempty"` // optional, if > 0 uses external versioning
	Body    json.RawMessage `json:"body"`
}

// BulkIndex sends a batch of documents to OpenSearch using the index action.
func BulkIndex(ctx context.Context, client *opensearchapi.Client, items []*Document) (int, []*Document, error) {
	if len(items) == 0 {
		return 0, nil, nil
	}

	var buf bytes.Buffer
	for _, item := range items {
		if item.Version > 0 {
			fmt.Fprintf(&buf, `{"index":{"_index":%q,"_id":%q,"routing":%q,"version":%d,"version_type":"external"}}`, item.Index, item.ID, item.Routing, item.Version)
		} else {
			fmt.Fprintf(&buf, `{"index":{"_index":%q,"_id":%q,"routing":%q}}`, item.Index, item.ID, item.Routing)
		}
		buf.WriteByte('\n')
		buf.Write(item.Body)
		buf.WriteByte('\n')
	}

	resp, err := client.Bulk(ctx, opensearchapi.BulkReq{Body: &buf})
	if err != nil {
		return 0, nil, fmt.Errorf("error sending bulk request to opensearch: %w", err)
	}

	if !resp.Errors {
		return len(items), nil, nil
	}

	numWritten := 0
	var retryable []*Document

	for i, itemMap := range resp.Items {
		for _, item := range itemMap {
			if item.Status >= 200 && item.Status < 300 {
				numWritten++
			} else if item.Status == 429 || item.Status >= 500 {
				retryable = append(retryable, items[i])
			} else if item.Status == 409 {
				slog.Debug("opensearch version conflict (ignored)", "index", items[i].Index, "id", items[i].ID, "version", items[i].Version)
			} else {
				errType, errReason := "", ""
				if item.Error != nil {
					errType = item.Error.Type
					errReason = item.Error.Reason
				}
				slog.Error("permanent opensearch bulk index failure", "index", items[i].Index, "status", item.Status, "error_type", errType, "error_reason", errReason)
			}
		}
	}

	return numWritten, retryable, nil
}
