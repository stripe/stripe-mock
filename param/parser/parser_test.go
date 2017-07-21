package parser

import (
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestParseForm(t *testing.T) {
	var err error
	var v formValues

	v, err = parseForm(``)
	assert.NoError(t, err)
	assert.Nil(t, v)

	v, err = parseForm(`a=b&c=d&e=f`)
	assert.NoError(t, err)
	assert.Equal(t, formValues{
		formPair{"a", "b"},
		formPair{"c", "d"},
		formPair{"e", "f"},
	}, v)

	v, err = parseForm(`a=b&a=c`)
	assert.NoError(t, err)
	assert.Equal(t, formValues{
		formPair{"a", "b"},
		formPair{"a", "c"},
	}, v)

	v, err = parseForm(`a=b&a=b`)
	assert.NoError(t, err)
	assert.Equal(t, formValues{
		formPair{"a", "b"},
		formPair{"a", "b"},
	}, v)

	v, err = parseForm(`?a=b&a=b`)
	assert.NoError(t, err)
	assert.Equal(t, formValues{
		formPair{"a", "b"},
		formPair{"a", "b"},
	}, v)

	v, err = parseForm(`?x=%20&%20=x`)
	assert.NoError(t, err)
	assert.Equal(t, formValues{
		formPair{"x", " "},
		formPair{" ", "x"},
	}, v)

	v, err = parseForm(`?x=+&+=x`)
	assert.NoError(t, err)
	assert.Equal(t, formValues{
		formPair{"x", " "},
		formPair{" ", "x"},
	}, v)

	v, err = parseForm(`?x=%2c&%2c=x`)
	assert.NoError(t, err)
	assert.Equal(t, formValues{
		formPair{"x", ","},
		formPair{",", "x"},
	}, v)

	_, err = parseForm(`%`)
	assert.Error(t, err)

	_, err = parseForm(`a=%`)
	assert.Error(t, err)
}

func TestParseKey(t *testing.T) {
	assert.Equal(t, []keyPart{
		&keyRaw{content: "name"},
	}, parseKey("name"))

	assert.Equal(t, []keyPart{
		&keyRaw{content: "array"},
		&keyArray{},
	}, parseKey("array[]"))

	assert.Equal(t, []keyPart{
		&keyRaw{content: "map"},
		&keyMap{content: "key"},
	}, parseKey("map[key]"))

	assert.Equal(t, []keyPart{
		&keyRaw{content: "maparray"},
		&keyArray{},
		&keyMap{content: "key"},
	}, parseKey("maparray[][key]"))

	assert.Equal(t, []keyPart{
		&keyRaw{content: "maparray"},
		&keyArray{},
		&keyMap{content: "key"},
		&keyArray{},
		&keyMap{content: "key"},
		&keyMap{content: "key"},
		&keyMap{content: "key"},
	}, parseKey("maparray[][key][][key][key][key]"))
}

func TestParseFormString(t *testing.T) {
	data, err := ParseFormString("foo=bar")
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"foo": "bar",
	}, data)
}

func TestParseFormValues_Basic(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"foo": "bar",
	}, mustParseFormValues(t, "foo=bar"))

	assert.Equal(t, map[string]interface{}{
		"foo": "7",
	}, mustParseFormValues(t, "foo=7"))

	assert.Equal(t, map[string]interface{}{
		"a": "value1",
		"b": "value2",
		"c": "value3",
	}, mustParseFormValues(t, "a=value1&b=value2&c=value3"))
}

func TestParseFormValues_Empty(t *testing.T) {
	assert.Equal(t, map[string]interface{}{}, mustParseFormValues(t, ""))
}

func TestParseFormValues_Array(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"arr": []interface{}{"value1"},
	}, mustParseFormValues(t, "arr[]=value1"))

	assert.Equal(t, map[string]interface{}{
		"arr": []interface{}{"value1", "value2", "value3"},
	}, mustParseFormValues(t, "arr[]=value1&arr[]=value2&arr[]=value3"))
}

// Here for completeness, but this kind of input is to a large degree nonsense
// and hopefully not present anywhere in the Stripe API ...
func TestParseFormValues_ArrayMulti(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"arr": []interface{}{
			[]interface{}{"value1"},
		},
	}, mustParseFormValues(t, "arr[][]=value1"))
}

func TestParseFormValues_Map(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"map": map[string]interface{}{"key1": "value1"},
	}, mustParseFormValues(t, "map[key1]=value1"))

	assert.Equal(t, map[string]interface{}{
		"map": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}, mustParseFormValues(t, "map[key1]=value1&map[key2]=value2"))

	assert.Equal(t, map[string]interface{}{
		"map1": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
		"map2": map[string]interface{}{
			"key1": "value1",
		},
	}, mustParseFormValues(t, "map1[key1]=value1&map1[key2]=value2&map2[key1]=value1"))
}

func TestParseFormValues_MapNesting(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"map": map[string]interface{}{
			"key1": map[string]interface{}{
				"key1": "value1_1",
				"key2": "value1_2",
			},
			"key2": "value2",
		},
	}, mustParseFormValues(t, "map[key1][key1]=value1_1&map[key1][key2]=value1_2&map[key2]=value2"))

	assert.Equal(t, map[string]interface{}{
		"map": map[string]interface{}{
			"key1": map[string]interface{}{
				"key1": map[string]interface{}{
					"key1": "value1",
				},
			},
			"key2": "value2",
		},
	}, mustParseFormValues(t, "map[key1][key1][key1]=value1&map[key2]=value2"))
}

func TestParseFormValues_MapInArray(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"obj": []interface{}{
			map[string]interface{}{"key1": "value1"},
		},
	}, mustParseFormValues(t, "obj[][key1]=value1"))

	assert.Equal(t, map[string]interface{}{
		"obj": []interface{}{
			map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}, mustParseFormValues(t, "obj[][key1]=value1&obj[][key2]=value2"))
}

func TestParseFormValues_MapInArrayMulti(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"obj": []interface{}{
			map[string]interface{}{
				"key1": "value1_1",
				"key2": "value1_2",
			},
			map[string]interface{}{
				"key1": "value2_1",
				"key2": "value2_2",
			},
		},
	}, mustParseFormValues(t,
		"obj[][key1]=value1_1&obj[][key2]=value1_2&"+
			"obj[][key1]=value2_1&obj[][key2]=value2_2",
	))
}

func TestParseFormValues_MapInArrayRepeating(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"obj": []interface{}{
			map[string]interface{}{
				"key1": "value1",
			},
			map[string]interface{}{
				"key1": "value2",
			},
		},
	}, mustParseFormValues(t, "obj[][key1]=value1&obj[][key1]=value2"))
}

func TestParseFormValues_ArrayInMap(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"obj": map[string]interface{}{
			"key1": []interface{}{"value1"},
		},
	}, mustParseFormValues(t, "obj[key1][]=value1"))

	assert.Equal(t, map[string]interface{}{
		"obj": map[string]interface{}{
			"key1": []interface{}{
				"value1",
				"value2",
			},
		},
	}, mustParseFormValues(t, "obj[key1][]=value1&obj[key1][]=value2"))
}

func TestParseFormValues_ComplexNesting(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"obj": []interface{}{
			map[string]interface{}{
				"key1": []interface{}{
					"value1_1",
					"value1_2",
				},
				"key2": map[string]interface{}{
					"key": "value2",
				},
				"key3": "value3",
			},
		},
	}, mustParseFormValues(t,
		"obj[][key1][]=value1_1&obj[][key1][]=value1_2&"+
			"obj[][key2][key]=value2&"+
			"obj[][key3]=value3",
	))
}

//
// ---
//

func mustParse(t *testing.T, query string) formValues {
	values, err := parseForm(query)
	assert.NoError(t, err)
	return values
}

func mustParseFormValues(t *testing.T, query string) map[string]interface{} {
	params, err := parseFormValues(mustParse(t, query))
	assert.NoError(t, err)
	return params
}
