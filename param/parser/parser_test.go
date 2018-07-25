package parser

import (
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/stripe/stripe-mock/param/form"
)

//
// Tests
//

func TestParseForm(t *testing.T) {
	var err error
	var v form.Values

	v, err = ParseFormString(``)
	assert.NoError(t, err)
	assert.Nil(t, v)

	v, err = ParseFormString(`a=b&c=d&e=f`)
	assert.NoError(t, err)
	assert.Equal(t, form.Values{
		form.Pair{"a", "b"},
		form.Pair{"c", "d"},
		form.Pair{"e", "f"},
	}, v)

	v, err = ParseFormString(`a=b&a=c`)
	assert.NoError(t, err)
	assert.Equal(t, form.Values{
		form.Pair{"a", "b"},
		form.Pair{"a", "c"},
	}, v)

	v, err = ParseFormString(`a=b&a=b`)
	assert.NoError(t, err)
	assert.Equal(t, form.Values{
		form.Pair{"a", "b"},
		form.Pair{"a", "b"},
	}, v)

	v, err = ParseFormString(`?a=b&a=b`)
	assert.NoError(t, err)
	assert.Equal(t, form.Values{
		form.Pair{"a", "b"},
		form.Pair{"a", "b"},
	}, v)

	v, err = ParseFormString(`?x=%20&%20=x`)
	assert.NoError(t, err)
	assert.Equal(t, form.Values{
		form.Pair{"x", " "},
		form.Pair{" ", "x"},
	}, v)

	v, err = ParseFormString(`?x=+&+=x`)
	assert.NoError(t, err)
	assert.Equal(t, form.Values{
		form.Pair{"x", " "},
		form.Pair{" ", "x"},
	}, v)

	v, err = ParseFormString(`?x=%2c&%2c=x`)
	assert.NoError(t, err)
	assert.Equal(t, form.Values{
		form.Pair{"x", ","},
		form.Pair{",", "x"},
	}, v)

	_, err = ParseFormString(`%`)
	assert.Error(t, err)

	_, err = ParseFormString(`a=%`)
	assert.Error(t, err)
}
