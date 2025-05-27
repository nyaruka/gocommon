package dynamo

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
)

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
