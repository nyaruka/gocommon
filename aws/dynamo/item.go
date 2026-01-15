package dynamo

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"time"
)

// Describes the common format for all items stored in DynamoDB.

// Key is the key type for all items, consisting of a partition key (PK) and a sort key (SK).
type Key struct {
	PK string `dynamodbav:"PK"`
	SK string `dynamodbav:"SK"`
}

func (k Key) String() string {
	return fmt.Sprintf("%s[%s]", k.PK, k.SK)
}

// Item is the common structure for items stored in DynamoDB.
type Item struct {
	Key

	OrgID  int            `dynamodbav:"OrgID"`
	TTL    *time.Time     `dynamodbav:"TTL,unixtime,omitempty"`
	Data   map[string]any `dynamodbav:"Data,omitempty"`
	DataGZ []byte         `dynamodbav:"DataGZ,omitempty"`
	Src    string         `dynamodbav:"Src,omitempty"`
}

func (i *Item) GetData() (map[string]any, error) {
	data := map[string]any{}

	if len(i.DataGZ) > 0 {
		if err := UnmarshalJSONGZ(i.DataGZ, &data); err != nil {
			return nil, err
		}
	}
	if len(i.Data) > 0 {
		maps.Copy(data, i.Data)
	}

	return data, nil
}

// MarshalJSON is only used for testing
func (i *Item) MarshalJSON() ([]byte, error) {
	var ttl *time.Time
	if i.TTL != nil {
		t := i.TTL.In(time.UTC)
		ttl = &t
	}

	return json.Marshal(struct {
		PK     string         `json:"PK"`
		SK     string         `json:"SK"`
		OrgID  int            `json:"OrgID"`
		TTL    *time.Time     `json:"TTL,omitempty"`
		Data   map[string]any `json:"Data"`
		DataGZ string         `json:"DataGZ,omitempty"`
		Src    string         `json:"Src,omitempty"`
	}{
		PK:     i.PK,
		SK:     i.SK,
		OrgID:  i.OrgID,
		TTL:    ttl,
		Data:   i.Data,
		DataGZ: base64.StdEncoding.EncodeToString(i.DataGZ),
		Src:    i.Src,
	})
}

func MarshalJSONGZ(v any) ([]byte, error) {
	buf := &bytes.Buffer{}
	w := gzip.NewWriter(buf)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		return nil, fmt.Errorf("error encoding value as json+gzip: %w", err)
	}

	w.Close()

	return buf.Bytes(), nil
}

func UnmarshalJSONGZ(d []byte, v any) error {
	r, err := gzip.NewReader(bytes.NewReader(d))
	if err != nil {
		return err
	}

	if err := json.NewDecoder(r).Decode(v); err != nil {
		return fmt.Errorf("error decoding value from json+gzip: %w", err)
	}

	return nil
}

type ItemMarshaler interface {
	MarshalDynamo() (*Item, error)
}
