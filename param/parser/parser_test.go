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
	var v form.FormValues

	v, err = ParseFormString(``)
	assert.NoError(t, err)
	assert.Nil(t, v)

	v, err = ParseFormString(`a=b&c=d&e=f`)
	assert.NoError(t, err)
	assert.Equal(t, form.FormValues{
		form.FormPair{"a", "b"},
		form.FormPair{"c", "d"},
		form.FormPair{"e", "f"},
	}, v)

	v, err = ParseFormString(`a=b&a=c`)
	assert.NoError(t, err)
	assert.Equal(t, form.FormValues{
		form.FormPair{"a", "b"},
		form.FormPair{"a", "c"},
	}, v)

	v, err = ParseFormString(`a=b&a=b`)
	assert.NoError(t, err)
	assert.Equal(t, form.FormValues{
		form.FormPair{"a", "b"},
		form.FormPair{"a", "b"},
	}, v)

	v, err = ParseFormString(`?a=b&a=b`)
	assert.NoError(t, err)
	assert.Equal(t, form.FormValues{
		form.FormPair{"a", "b"},
		form.FormPair{"a", "b"},
	}, v)

	v, err = ParseFormString(`?x=%20&%20=x`)
	assert.NoError(t, err)
	assert.Equal(t, form.FormValues{
		form.FormPair{"x", " "},
		form.FormPair{" ", "x"},
	}, v)

	v, err = ParseFormString(`?x=+&+=x`)
	assert.NoError(t, err)
	assert.Equal(t, form.FormValues{
		form.FormPair{"x", " "},
		form.FormPair{" ", "x"},
	}, v)

	v, err = ParseFormString(`?x=%2c&%2c=x`)
	assert.NoError(t, err)
	assert.Equal(t, form.FormValues{
		form.FormPair{"x", ","},
		form.FormPair{",", "x"},
	}, v)

	_, err = ParseFormString(`%`)
	assert.Error(t, err)

	_, err = ParseFormString(`a=%`)
	assert.Error(t, err)
}
