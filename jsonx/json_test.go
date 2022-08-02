package jsonx_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/nyaruka/gocommon/jsonx"

	"github.com/stretchr/testify/assert"
)

type badThing struct{}

func (b *badThing) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("I can't be marshaled")
}

func (b *badThing) UnmarshalJSON() error {
	return fmt.Errorf("I can't be unmarshaled")
}

func TestMarshaling(t *testing.T) {
	j, err := jsonx.Marshal(nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte(`null`), j)

	// check that HTML entities aren't encoded
	j, err = jsonx.Marshal("Rwanda > Kigali & Ecuador")
	assert.NoError(t, err)
	assert.Equal(t, []byte(`"Rwanda > Kigali & Ecuador"`), j)

	// check an object that can be marshaled
	j, err = jsonx.Marshal(map[string]string{"foo": "bar"})
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"foo":"bar"}`), j)

	// and one that can't
	_, err = jsonx.Marshal(&badThing{})
	assert.EqualError(t, err, "json: error calling MarshalJSON for type *jsonx_test.badThing: I can't be marshaled")

	j, err = jsonx.MarshalPretty(map[string]string{"foo": "bar"})
	assert.NoError(t, err)
	assert.Equal(t, []byte("{\n    \"foo\": \"bar\"\n}"), j)

	j, err = jsonx.MarshalMerged(map[string]string{"foo": "bar"}, map[string]string{"zed": "xyz"})
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"foo":"bar","zed":"xyz"}`), j)
}

func TestMustMarshal(t *testing.T) {
	// check an object that can be marshaled
	j := jsonx.MustMarshal(map[string]string{"foo": "bar"})
	assert.Equal(t, []byte(`{"foo":"bar"}`), j)

	// and one that can't
	assert.PanicsWithError(t, "json: error calling MarshalJSON for type *jsonx_test.badThing: I can't be marshaled", func() {
		jsonx.MustMarshal(&badThing{})
	})
}

func TestUnmarshal(t *testing.T) {
	var s string
	err := jsonx.Unmarshal([]byte(`"test"`), &s)
	assert.NoError(t, err)
	assert.Equal(t, "test", s)

	var b badThing
	err = jsonx.Unmarshal([]byte(`"test"`), &b)
	assert.EqualError(t, err, "json: cannot unmarshal string into Go value of type jsonx_test.badThing")
}

func TestMustUnmarshal(t *testing.T) {
	var s string
	jsonx.MustUnmarshal([]byte(`"test"`), &s)
	assert.Equal(t, "test", s)

	assert.PanicsWithError(t, "json: cannot unmarshal string into Go value of type jsonx_test.badThing", func() {
		var b badThing
		jsonx.MustUnmarshal([]byte(`"test"`), &b)
	})
}

func TestUnmarshalArray(t *testing.T) {
	// test empty array
	msgs, err := jsonx.UnmarshalArray([]byte(`[]`))
	assert.NoError(t, err)
	assert.Equal(t, []json.RawMessage{}, msgs)
}

func TestUnmarshalWithLimit(t *testing.T) {
	data := []byte(`{"foo": "Hello"}`)
	buffer := ioutil.NopCloser(bytes.NewReader(data))

	// try with sufficiently large limit
	s := &struct {
		Foo string `json:"foo"`
	}{}
	err := jsonx.UnmarshalWithLimit(buffer, s, 1000)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", s.Foo)

	// try with limit that's smaller than the input
	buffer = ioutil.NopCloser(bytes.NewReader(data))
	s = &struct {
		Foo string `json:"foo"`
	}{}
	err = jsonx.UnmarshalWithLimit(buffer, s, 5)
	assert.EqualError(t, err, "unexpected end of JSON input")
}

func TestDecodeGeneric(t *testing.T) {
	// parse a JSON object into a map
	data := []byte(`{"bool": true, "number": 123.34, "text": "hello", "object": {"foo": "bar"}, "array": [1, "x"]}`)
	vals, err := jsonx.DecodeGeneric(data)
	assert.NoError(t, err)

	asMap := vals.(map[string]interface{})
	assert.Equal(t, true, asMap["bool"])
	assert.Equal(t, json.Number("123.34"), asMap["number"])
	assert.Equal(t, "hello", asMap["text"])
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, asMap["object"])
	assert.Equal(t, []interface{}{json.Number("1"), "x"}, asMap["array"])

	// parse a JSON array into a slice
	data = []byte(`[{"foo": 123}, {"foo": 456}]`)
	vals, err = jsonx.DecodeGeneric(data)
	assert.NoError(t, err)

	asSlice := vals.([]interface{})
	assert.Equal(t, map[string]interface{}{"foo": json.Number("123")}, asSlice[0])
	assert.Equal(t, map[string]interface{}{"foo": json.Number("456")}, asSlice[1])
}
