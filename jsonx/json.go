package jsonx

import (
	"bytes"
	"encoding/json"
	"io"
)

// Marshal marshals the given object to JSON
func Marshal(v any) ([]byte, error) {
	return marshal(v, "")
}

// MarshalPretty marshals the given object to pretty JSON
func MarshalPretty(v any) ([]byte, error) {
	return marshal(v, "    ")
}

// MarshalMerged marshals the properties of two objects as one object
func MarshalMerged(v1 any, v2 any) ([]byte, error) {
	b1, err := marshal(v1, "")
	if err != nil {
		return nil, err
	}
	b2, err := marshal(v2, "")
	if err != nil {
		return nil, err
	}
	b := append(b1[0:len(b1)-1], byte(','))
	b = append(b, b2[1:]...)
	return b, nil
}

// MustMarshal marshals the given object to JSON, panicking on an error
func MustMarshal(v any) []byte {
	data, err := marshal(v, "")
	if err != nil {
		panic(err)
	}
	return data
}

func marshal(v any, indent string) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false) // see https://github.com/golang/go/issues/8592
	encoder.SetIndent("", indent)

	err := encoder.Encode(v)
	if err != nil {
		return nil, err
	}

	// don't include the final \n that .Encode() adds
	data := buffer.Bytes()
	return data[0 : len(data)-1], nil
}

// Unmarshal is just a shortcut for json.Unmarshal so all calls can be made via the jsonx package
func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// UnmarshalArray unmarshals an array of objects from the given JSON
func UnmarshalArray(data []byte) ([]json.RawMessage, error) {
	var items []json.RawMessage
	err := Unmarshal(data, &items)
	return items, err
}

// UnmarshalWithLimit unmarsmals a struct with a limit on how many bytes can be read from the given reader
func UnmarshalWithLimit(reader io.ReadCloser, s any, limit int64) error {
	body, err := io.ReadAll(io.LimitReader(reader, limit))
	if err != nil {
		return err
	}
	if err := reader.Close(); err != nil {
		return err
	}
	return Unmarshal(body, &s)
}

// MustUnmarshal unmarshals the given JSON, panicking on an error
func MustUnmarshal(data []byte, v any) {
	if err := json.Unmarshal(data, v); err != nil {
		panic(err)
	}
}

// DecodeGeneric decodes the given JSON as a generic map or slice
func DecodeGeneric(data []byte) (any, error) {
	var asGeneric any
	decoder := json.NewDecoder(bytes.NewBuffer(data))
	decoder.UseNumber()
	return asGeneric, decoder.Decode(&asGeneric)
}
