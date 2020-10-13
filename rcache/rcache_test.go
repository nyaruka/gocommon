package rcache

import (
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
)

func TestMarker(t *testing.T) {
	tcs := []struct {
		Action string
		Group  string
		Key    string
		Value  string
	}{
		{"clear", "wa", "", ""},
		{"clear", "tel", "", ""},
		{"get", "wa", "foo", ""},
		{"set", "wa", "foo", "bar"},
		{"get", "wa", "foo", "bar"},
		{"get", "tel", "foo", ""},
		{"set", "tel", "foo", "baz"},
		{"get", "tel", "foo", "baz"},
		{"delete", "wa", "foo", ""},
		{"get", "wa", "foo", ""},
		{"get", "tel", "foo", "baz"},
		{"clear", "tel", "", ""},
		{"get", "tel", "foo", ""},
	}

	rc, err := redis.Dial("tcp", "localhost:6379")
	if err != nil {
		panic(err)
	}
	_, err = rc.Do("SELECT", 0)
	if err != nil {
		panic(err)
	}
	defer rc.Close()

	for i, tc := range tcs {
		if tc.Action == "set" {
			err := Set(rc, tc.Group, tc.Key, tc.Value)
			assert.NoError(t, err)
		} else if tc.Action == "get" {
			value, err := Get(rc, tc.Group, tc.Key)
			assert.NoError(t, err)
			assert.Equal(t, tc.Value, value, "%d: not equal", i)
		} else if tc.Action == "delete" {
			err := Delete(rc, tc.Group, tc.Key)
			assert.NoError(t, err)
		} else if tc.Action == "clear" {
			err := Clear(rc, tc.Group)
			assert.NoError(t, err)
		}
	}
}
