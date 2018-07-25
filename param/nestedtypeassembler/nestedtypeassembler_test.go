package nestedtypeassembler

import (
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/stripe/stripe-mock/param/form"
	"github.com/stripe/stripe-mock/param/parser"
)

//
// Tests
//

func TestAssembleParams_Basic(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"foo": "bar",
	}, mustAssembleParams(t, "foo=bar"))

	assert.Equal(t, map[string]interface{}{
		"foo": "7",
	}, mustAssembleParams(t, "foo=7"))

	assert.Equal(t, map[string]interface{}{
		"a": "value1",
		"b": "value2",
		"c": "value3",
	}, mustAssembleParams(t, "a=value1&b=value2&c=value3"))
}

func TestAssembleParams_Empty(t *testing.T) {
	assert.Equal(t, map[string]interface{}{}, mustAssembleParams(t, ""))
}

func TestAssembleParams_Array(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"arr": []interface{}{"value1"},
	}, mustAssembleParams(t, "arr[]=value1"))

	assert.Equal(t, map[string]interface{}{
		"arr": []interface{}{"value1", "value2", "value3"},
	}, mustAssembleParams(t, "arr[]=value1&arr[]=value2&arr[]=value3"))
}

// Here for completeness, but this kind of input is to a large degree nonsense
// and hopefully not present anywhere in the Stripe API ...
func TestAssembleParams_ArrayMulti(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"arr": []interface{}{
			[]interface{}{"value1"},
		},
	}, mustAssembleParams(t, "arr[][]=value1"))
}

func TestAssembleParams_Map(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"map": map[string]interface{}{"key1": "value1"},
	}, mustAssembleParams(t, "map[key1]=value1"))

	assert.Equal(t, map[string]interface{}{
		"map": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}, mustAssembleParams(t, "map[key1]=value1&map[key2]=value2"))

	assert.Equal(t, map[string]interface{}{
		"map1": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
		"map2": map[string]interface{}{
			"key1": "value1",
		},
	}, mustAssembleParams(t, "map1[key1]=value1&map1[key2]=value2&map2[key1]=value1"))
}

func TestAssembleParams_MapNesting(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"map": map[string]interface{}{
			"key1": map[string]interface{}{
				"key1": "value1_1",
				"key2": "value1_2",
			},
			"key2": "value2",
		},
	}, mustAssembleParams(t, "map[key1][key1]=value1_1&map[key1][key2]=value1_2&map[key2]=value2"))

	assert.Equal(t, map[string]interface{}{
		"map": map[string]interface{}{
			"key1": map[string]interface{}{
				"key1": map[string]interface{}{
					"key1": "value1",
				},
			},
			"key2": "value2",
		},
	}, mustAssembleParams(t, "map[key1][key1][key1]=value1&map[key2]=value2"))
}

func TestAssembleParams_MapInArray(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"obj": []interface{}{
			map[string]interface{}{"key1": "value1"},
		},
	}, mustAssembleParams(t, "obj[][key1]=value1"))

	assert.Equal(t, map[string]interface{}{
		"obj": []interface{}{
			map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}, mustAssembleParams(t, "obj[][key1]=value1&obj[][key2]=value2"))
}

func TestAssembleParams_MapInArrayMulti(t *testing.T) {
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
	}, mustAssembleParams(t,
		"obj[][key1]=value1_1&obj[][key2]=value1_2&"+
			"obj[][key1]=value2_1&obj[][key2]=value2_2",
	))
}

func TestAssembleParams_MapInArrayRepeating(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"obj": []interface{}{
			map[string]interface{}{
				"key1": "value1",
			},
			map[string]interface{}{
				"key1": "value2",
			},
		},
	}, mustAssembleParams(t, "obj[][key1]=value1&obj[][key1]=value2"))
}

func TestAssembleParams_ArrayInMap(t *testing.T) {
	assert.Equal(t, map[string]interface{}{
		"obj": map[string]interface{}{
			"key1": []interface{}{"value1"},
		},
	}, mustAssembleParams(t, "obj[key1][]=value1"))

	assert.Equal(t, map[string]interface{}{
		"obj": map[string]interface{}{
			"key1": []interface{}{
				"value1",
				"value2",
			},
		},
	}, mustAssembleParams(t, "obj[key1][]=value1&obj[key1][]=value2"))
}

func TestAssembleParams_ComplexNesting(t *testing.T) {
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
	}, mustAssembleParams(t,
		"obj[][key1][]=value1_1&obj[][key1][]=value1_2&"+
			"obj[][key2][key]=value2&"+
			"obj[][key3]=value3",
	))
}

//
// Tests for private functions
//

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

//
// Private functions
//

func mustParse(t *testing.T, query string) form.Values {
	values, err := parser.ParseFormString(query)
	assert.NoError(t, err)
	return values
}

func mustAssembleParams(t *testing.T, query string) map[string]interface{} {
	params, err := AssembleParams(mustParse(t, query))
	assert.NoError(t, err)
	return params
}
