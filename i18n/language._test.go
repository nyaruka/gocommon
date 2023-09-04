package i18n_test

import (
	"testing"

	"github.com/nyaruka/gocommon/i18n"

	"github.com/stretchr/testify/assert"
)

func TestLanguage(t *testing.T) {
	lang, err := i18n.ParseLanguage("ENG")
	assert.NoError(t, err)
	assert.Equal(t, i18n.Language("eng"), lang)

	_, err = i18n.ParseLanguage("base")
	assert.EqualError(t, err, "iso-639-3 codes must be 3 characters, got: base")

	_, err = i18n.ParseLanguage("xzx")
	assert.EqualError(t, err, "unrecognized language code: xzx")

	v, err := i18n.Language("eng").Value()
	assert.NoError(t, err)
	assert.Equal(t, "eng", v)

	v, err = i18n.NilLanguage.Value()
	assert.NoError(t, err)
	assert.Nil(t, v)

	var l i18n.Language
	assert.NoError(t, l.Scan("eng"))
	assert.Equal(t, i18n.Language("eng"), l)

	assert.NoError(t, l.Scan(nil))
	assert.Equal(t, i18n.NilLanguage, l)
}
