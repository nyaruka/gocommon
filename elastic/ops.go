package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/elastic/go-elasticsearch/v8"
)

// Document is a document to be indexed in Elasticsearch.
type Document struct {
	Index   string          `json:"index"`
	ID      string          `json:"id"`
	Routing string          `json:"routing"`
	Version int64           `json:"version,omitempty"` // optional, if > 0 uses external versioning
	Body    json.RawMessage `json:"body"`
}

// BulkIndex sends a batch of documents to Elasticsearch using the index action.
func BulkIndex(ctx context.Context, client *elasticsearch.Client, items []*Document) (int, []*Document, error) {
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

	resp, err := client.Bulk(&buf, client.Bulk.WithContext(ctx))
	if err != nil {
		return 0, nil, fmt.Errorf("error sending bulk request to elasticsearch: %w", err)
	}
	defer resp.Body.Close()

	// if we got a non-2xx response, the entire request failed (e.g. 413 Request Entity Too Large)
	// and the response body won't contain per-item results
	if resp.IsError() {
		return 0, items, fmt.Errorf("elasticsearch bulk request failed with status %d", resp.StatusCode)
	}

	var result bulkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, items, fmt.Errorf("error decoding elasticsearch bulk response: %w", err)
	}

	if !result.Errors {
		return len(items), nil, nil
	}

	numWritten := 0
	var unprocessed []*Document

	for i, itemMap := range result.Items {
		for _, item := range itemMap {
			if item.Status >= 200 && item.Status < 300 {
				numWritten++
			} else if item.Status == 409 {
				slog.Debug("elasticsearch version conflict (ignored)", "index", items[i].Index, "id", items[i].ID, "version", items[i].Version)
			} else {
				if item.Status == 429 || item.Status >= 500 {
					slog.Error("retryable elasticsearch bulk index failure", "index", items[i].Index, "status", item.Status)
				} else {
					errType, errReason := "", ""
					if item.Error != nil {
						errType = item.Error.Type
						errReason = item.Error.Reason
					}
					slog.Error("permanent elasticsearch bulk index failure", "index", items[i].Index, "status", item.Status, "error_type", errType, "error_reason", errReason)
				}
				unprocessed = append(unprocessed, items[i])
			}
		}
	}

	return numWritten, unprocessed, nil
}

type bulkResponse struct {
	Errors bool                     `json:"errors"`
	Items  []map[string]bulkAction  `json:"items"`
}

type bulkAction struct {
	Status int        `json:"status"`
	Error  *bulkError `json:"error,omitempty"`
}

type bulkError struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}
