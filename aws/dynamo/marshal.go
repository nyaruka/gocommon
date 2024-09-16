package dynamo

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Marshaler interface {
	MarshalDynamo() (map[string]types.AttributeValue, error)
}

type Unmarshaler interface {
	UnmarshalDynamo(map[string]types.AttributeValue) error
}

func marshal(v any) (map[string]types.AttributeValue, error) {
	marshaler, ok := v.(Marshaler)
	if ok {
		return marshaler.MarshalDynamo()
	}

	return attributevalue.MarshalMap(v)
}

func unmarshal(m map[string]types.AttributeValue, v any) error {
	unmarshaler, ok := v.(Unmarshaler)
	if ok {
		return unmarshaler.UnmarshalDynamo(m)
	}

	return attributevalue.UnmarshalMap(m, v)
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
