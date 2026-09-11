package jsonx

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io"
)

// options passed to all marshaling and unmarshaling calls: v1 semantics (so struct tags and leniency behave as they
// always have) except for HTML escaping which this package has always disabled. As our apps are audited for v2's
// stricter defaults and changed tag semantics, options can be removed here until this is empty.
var compatOptions = json.JoinOptions(jsonv1.DefaultOptionsV1(), jsontext.EscapeForHTML(false))

// Marshal marshals the given object to JSON
func Marshal(v any) ([]byte, error) {
	return json.Marshal(v, compatOptions)
}

// MarshalPretty marshals the given object to pretty JSON
func MarshalPretty(v any) ([]byte, error) {
	return json.Marshal(v, compatOptions, jsontext.WithIndent("    "))
}

// MarshalMerged marshals the properties of two objects as one object
func MarshalMerged(v1 any, v2 any) ([]byte, error) {
	b1, err := Marshal(v1)
	if err != nil {
		return nil, err
	}
	b2, err := Marshal(v2)
	if err != nil {
		return nil, err
	}
	b := append(b1[0:len(b1)-1], byte(','))
	b = append(b, b2[1:]...)
	return b, nil
}

// MustMarshal marshals the given object to JSON, panicking on an error
func MustMarshal(v any) []byte {
	data, err := Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// Unmarshal unmarshals the given JSON into the given object
func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v, compatOptions)
}

// UnmarshalWithLimit unmarshals from the given reader with a limit on how many bytes can be read. The reader is
// always closed, and if unmarshaling succeeded, an error closing it is returned.
func UnmarshalWithLimit(reader io.ReadCloser, v any, limit int64) error {
	err := json.UnmarshalRead(io.LimitReader(reader, limit), v, compatOptions)
	if cerr := reader.Close(); err == nil {
		err = cerr
	}
	return err
}

// MustUnmarshal unmarshals the given JSON, panicking on an error
func MustUnmarshal(data []byte, v any) {
	if err := Unmarshal(data, v); err != nil {
		panic(err)
	}
}
