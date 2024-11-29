package i18n_test

import (
	"testing"

	"github.com/nyaruka/gocommon/i18n"
	"github.com/stretchr/testify/assert"
)

func TestDeriveCountryFromTel(t *testing.T) {
	assert.Equal(t, i18n.Country("RW"), i18n.DeriveCountryFromTel("+250788383383"))
	assert.Equal(t, i18n.Country("EC"), i18n.DeriveCountryFromTel("+593979000000"))

	assert.Equal(t, i18n.NilCountry, i18n.DeriveCountryFromTel("+80000000000")) // ignore 001
	assert.Equal(t, i18n.NilCountry, i18n.DeriveCountryFromTel("1234"))

	v, err := i18n.Country("RW").Value()
	assert.NoError(t, err)
	assert.Equal(t, "RW", v)

	v, err = i18n.NilCountry.Value()
	assert.NoError(t, err)
	assert.Nil(t, v)

	var c i18n.Country
	assert.NoError(t, c.Scan("RW"))
	assert.Equal(t, i18n.Country("RW"), c)

	assert.NoError(t, c.Scan(nil))
	assert.Equal(t, i18n.NilCountry, c)
}
