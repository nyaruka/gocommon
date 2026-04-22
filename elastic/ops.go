package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/versiontype"
)

// Action is the bulk operation to perform on a document.
type Action string

const (
	ActionIndex  Action = "index"
	ActionDelete Action = "delete"
)

// Document is a document to be indexed in or deleted from Elasticsearch.
type Document struct {
	Action  Action          `json:"action,omitempty"`  // defaults to index
	Index   string          `json:"index"`
	ID      string          `json:"id"`
	Routing string          `json:"routing"`
	Version int64           `json:"version,omitempty"` // optional, if > 0 uses external versioning
	Body    json.RawMessage `json:"body,omitempty"`    // required for index, ignored for delete
}

// Bulk sends a batch of documents to Elasticsearch, performing the action specified on each one.
func Bulk(ctx context.Context, client *elasticsearch.TypedClient, items []*Document) (int, []*Document, error) {
	if len(items) == 0 {
		return 0, nil, nil
	}

	req := client.Bulk()
	for _, item := range items {
		switch item.Action {
		case ActionDelete:
			op := types.DeleteOperation{
				Index_:  &item.Index,
				Id_:     &item.ID,
				Routing: &item.Routing,
			}
			if item.Version > 0 {
				op.Version = &item.Version
				vt := versiontype.External
				op.VersionType = &vt
			}
			if err := req.DeleteOp(op); err != nil {
				return 0, items, fmt.Errorf("error building bulk delete operation: %w", err)
			}
		default: // "" or ActionIndex
			op := types.IndexOperation{
				Index_:  &item.Index,
				Id_:     &item.ID,
				Routing: &item.Routing,
			}
			if item.Version > 0 {
				op.Version = &item.Version
				vt := versiontype.External
				op.VersionType = &vt
			}
			if err := req.IndexOp(op, item.Body); err != nil {
				return 0, items, fmt.Errorf("error building bulk index operation: %w", err)
			}
		}
	}

	resp, err := req.Do(ctx)
	if err != nil {
		// if the entire request failed (e.g. 413 Request Entity Too Large), all items are unprocessed
		var esErr *types.ElasticsearchError
		if errors.As(err, &esErr) {
			return 0, items, fmt.Errorf("elasticsearch bulk request failed with status %d: %w", esErr.Status, err)
		}
		return 0, nil, fmt.Errorf("error sending bulk request to elasticsearch: %w", err)
	}

	if !resp.Errors {
		return len(items), nil, nil
	}

	numWritten := 0
	var unprocessed []*Document

	for i, itemMap := range resp.Items {
		for _, item := range itemMap {
			if item.Status >= 200 && item.Status < 300 {
				numWritten++
			} else if item.Status == 409 {
				slog.Debug("elasticsearch version conflict (ignored)", "index", items[i].Index, "id", items[i].ID, "version", items[i].Version)
			} else if item.Status == 404 && items[i].Action == ActionDelete {
				slog.Debug("elasticsearch delete of missing document (ignored)", "index", items[i].Index, "id", items[i].ID)
			} else {
				if item.Status == 429 || item.Status >= 500 {
					slog.Error("retryable elasticsearch bulk failure", "index", items[i].Index, "action", items[i].Action, "status", item.Status)
				} else {
					errType, errReason := "", ""
					if item.Error != nil {
						errType = item.Error.Type
						if item.Error.Reason != nil {
							errReason = *item.Error.Reason
						}
					}
					slog.Error("permanent elasticsearch bulk failure", "index", items[i].Index, "action", items[i].Action, "status", item.Status, "error_type", errType, "error_reason", errReason)
				}
				unprocessed = append(unprocessed, items[i])
			}
		}
	}

	return numWritten, unprocessed, nil
}
