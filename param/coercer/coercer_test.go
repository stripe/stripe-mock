package coercer

import (
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/stripe/stripe-mock/spec"
)

func TestCoerceParams_AnyOfCoercion(t *testing.T) {
	// `anyOf` with basic types
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"bool_or_int_key": {
				AnyOf: []*spec.Schema{
					{Type: booleanType},
					{Type: integerType},
				},
			},
		}}
		data := map[string]interface{}{
			"bool_or_int_key": "123",
		}

		err := CoerceParams(schema, data)
		assert.NoError(t, err)
		assert.Equal(t, 123, data["bool_or_int_key"])
	}

	// `anyOf` with object type
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"object_or_int_key": {
				AnyOf: []*spec.Schema{
					{Properties: map[string]*spec.Schema{
						"intkey": {Type: integerType},
					}},
					{Type: integerType},
				},
			},
		}}
		data := map[string]interface{}{
			"object_or_int_key": map[string]interface{}{
				"intkey": "123",
			},
		}

		err := CoerceParams(schema, data)
		assert.NoError(t, err)
		assert.Equal(t, 123, data["object_or_int_key"].(map[string]interface{})["intkey"])
	}

	// `anyOf` with array of mixed types
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"array_object_or_int_key": {
				AnyOf: []*spec.Schema{
					// primitives are skipped
					{Type: integerType},
					{Type: stringType},
					// array of int is skipped
					{Items: &spec.Schema{
						Type: integerType,
					}, Type: arrayType},
					// array of object is chosen
					{Items: &spec.Schema{
						Properties: map[string]*spec.Schema{
							"intkey": {Type: integerType},
						}},
						Type: arrayType},
				},
			},
		}}
		data := map[string]interface{}{
			"array_object_or_int_key": map[string]interface{}{
				"0": map[string]interface{}{"intkey": "123"},
				"1": map[string]interface{}{"intkey": nil},
				"2": map[string]interface{}{"intkey": "124"},
			},
		}

		err := CoerceParams(schema, data)
		assert.NoError(t, err)

		sliceVal, ok := data["array_object_or_int_key"].([]interface{})
		assert.True(t, ok)

		assert.Equal(t, 123, sliceVal[0].(map[string]interface{})["intkey"])
		assert.Equal(t, nil, sliceVal[1].(map[string]interface{})["intkey"])
		assert.Equal(t, 124, sliceVal[2].(map[string]interface{})["intkey"])
	}

	//`anyOf` with array of different primitives
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"array_string_or_array_int": {
				AnyOf: []*spec.Schema{
					{Type: booleanType},
					// array of string is chosen
					{Items: &spec.Schema{
						Type: stringType},
						Type: arrayType},
					{Items: &spec.Schema{
						Type: integerType,
					},
						Type: arrayType},
				},
			},
		}}
		data := map[string]interface{}{
			"array_string_or_array_int": map[string]interface{}{
				"0": "123",
				"1": nil,
				"2": "456",
			},
		}

		err := CoerceParams(schema, data)
		assert.NoError(t, err)

		sliceVal, ok := data["array_string_or_array_int"].([]interface{})
		assert.True(t, ok)

		// values are not coerced into numbers, given explicit string-typed
		assert.Equal(t, "123", sliceVal[0])
		assert.Equal(t, nil, sliceVal[1])
		assert.Equal(t, "456", sliceVal[2])
	}

	// `anyOf` Enum and numeric string
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"enum_or_int_key": {
				AnyOf: []*spec.Schema{
					{Enum: []interface{}{
						"enum1",
						"enum2"},
						Type: stringType},
					{Type: integerType},
				},
			},
		}}

		// numeric string shouldn't be considered as enum string
		// and still be coerced to number
		data := map[string]interface{}{
			"enum_or_int_key": "123",
		}
		err := CoerceParams(schema, data)
		assert.NoError(t, err)
		assert.Equal(t, 123, data["enum_or_int_key"])

		// note that value "foo" is not a valid value, but
		// validation is not done here. having "foo" as string is sufficient
		data = map[string]interface{}{
			"enum_or_int_key": "foo",
		}
		err = CoerceParams(schema, data)
		assert.NoError(t, err)
		assert.Equal(t, "foo", data["enum_or_int_key"])
	}
}

func TestCoerceParams_ArrayCoercion(t *testing.T) {
	// Array of primitive values
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"arraykey": {
				Items: &spec.Schema{
					Type: integerType,
				},
				Type: arrayType,
			},
		}}
		data := map[string]interface{}{
			"arraykey": []interface{}{
				"123",
				nil,
				"124",
			},
		}

		err := CoerceParams(schema, data)
		assert.NoError(t, err)

		sliceVal, ok := data["arraykey"].([]interface{})
		assert.True(t, ok)

		assert.Equal(t, 123, sliceVal[0])
		assert.Equal(t, nil, sliceVal[1])
		assert.Equal(t, 124, sliceVal[2])
	}

	// Array of objects
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"arraykey": {
				Items: &spec.Schema{
					Properties: map[string]*spec.Schema{
						"intkey": {Type: integerType},
					},
				},
				Type: arrayType,
			},
		}}
		data := map[string]interface{}{
			"arraykey": []interface{}{
				map[string]interface{}{"intkey": "123"},
				map[string]interface{}{"intkey": nil},
				map[string]interface{}{"intkey": "124"},
			},
		}

		err := CoerceParams(schema, data)
		assert.NoError(t, err)

		sliceVal, ok := data["arraykey"].([]interface{})
		assert.True(t, ok)

		assert.Equal(t, 123, sliceVal[0].(map[string]interface{})["intkey"])
		assert.Equal(t, nil, sliceVal[1].(map[string]interface{})["intkey"])
		assert.Equal(t, 124, sliceVal[2].(map[string]interface{})["intkey"])
	}

	// Integer-indexed map array
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"arraykey": {
				Items: &spec.Schema{
					Properties: map[string]*spec.Schema{
						"intkey": {Type: integerType},
					},
				},
				Type: arrayType,
			},
		}}
		data := map[string]interface{}{
			"arraykey": map[string]interface{}{
				"0": map[string]interface{}{"intkey": "123"},
				"1": map[string]interface{}{"intkey": nil},
				"2": map[string]interface{}{"intkey": "124"},
			},
		}

		err := CoerceParams(schema, data)
		assert.NoError(t, err)

		sliceVal, ok := data["arraykey"].([]interface{})
		assert.True(t, ok)

		assert.Equal(t, 123, sliceVal[0].(map[string]interface{})["intkey"])
		assert.Equal(t, nil, sliceVal[1].(map[string]interface{})["intkey"])
		assert.Equal(t, 124, sliceVal[2].(map[string]interface{})["intkey"])
	}

	// Integer-indexed of enum
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"array_enum": {
				Items: &spec.Schema{
					Enum: []interface{}{
						"enum1",
						"enum2",
					}, Type: stringType,
				},
				Type: arrayType,
			},
		}}
		data := map[string]interface{}{
			"array_enum": map[string]interface{}{
				// enum value is incorrect, but only coercion
				// to string is required
				"0": "enum",
			},
		}

		err := CoerceParams(schema, data)
		assert.NoError(t, err)

		sliceVal, ok := data["array_enum"].([]interface{})
		assert.True(t, ok)

		assert.Equal(t, "enum", sliceVal[0])
	}
}

func TestCoerceParams_BooleanCoercion(t *testing.T) {
	schema := &spec.Schema{Properties: map[string]*spec.Schema{
		"boolkey": {Type: booleanType},
	}}
	data := map[string]interface{}{
		"boolkey": "true",
	}

	err := CoerceParams(schema, data)
	assert.NoError(t, err)
	assert.Equal(t, true, data["boolkey"])
}

func TestCoerceParams_IntegerCoercion(t *testing.T) {
	schema := &spec.Schema{Properties: map[string]*spec.Schema{
		"intkey": {Type: integerType},
	}}
	data := map[string]interface{}{
		"intkey": "123",
	}

	err := CoerceParams(schema, data)
	assert.NoError(t, err)
	assert.Equal(t, 123, data["intkey"])
}

func TestCoerceParams_IntegerIndexedMapCoercion(t *testing.T) {
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"arraykey": {Type: arrayType},
		}}
		data := map[string]interface{}{
			"arraykey": map[string]interface{}{
				"0": "0-index",
				"2": "2-index",
			},
		}

		err := CoerceParams(schema, data)
		assert.NoError(t, err)

		sliceVal, ok := data["arraykey"].([]interface{})
		assert.True(t, ok)

		assert.Equal(t, "0-index", sliceVal[0])
		assert.Equal(t, nil, sliceVal[1])
		assert.Equal(t, "2-index", sliceVal[2])
	}

	// Value was not a map
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"arraykey": {Type: arrayType},
		}}
		data := map[string]interface{}{
			"arraykey": "not-map",
		}

		err := CoerceParams(schema, data)
		assert.NoError(t, err)
		assert.Equal(t, "not-map", data["arraykey"])
	}

	// Not all indexes were integers
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"arraykey": {Type: arrayType},
		}}
		data := map[string]interface{}{
			"arraykey": map[string]interface{}{
				"0":   "0-index",
				"foo": "foo-index",
			},
		}

		err := CoerceParams(schema, data)
		assert.NoError(t, err)
		assert.Equal(t, "foo-index", data["arraykey"].(map[string]interface{})["foo"])
	}

	// Index too big
	{
		schema := &spec.Schema{Properties: map[string]*spec.Schema{
			"arraykey": {Type: arrayType},
		}}
		data := map[string]interface{}{
			"arraykey": map[string]interface{}{
				"999999": "big-index",
			},
		}

		err := CoerceParams(schema, data)
		assert.Error(t, err)
	}
}

func TestCoerceParams_NumberCoercion(t *testing.T) {
	schema := &spec.Schema{Properties: map[string]*spec.Schema{
		"numberkey": {Type: numberType},
	}}
	data := map[string]interface{}{
		"numberkey": "123.45",
	}

	err := CoerceParams(schema, data)
	assert.NoError(t, err)
	assert.Equal(t, 123.45, data["numberkey"])
}

func TestCoerceParams_UnspecifiedTypeNoCoercion(t *testing.T) {
	schema := &spec.Schema{Properties: map[string]*spec.Schema{
		// unspecified types
		"stringkey": {},
		"numberKey": {},
		"objectKey": {},
	}}
	data := map[string]interface{}{
		"stringkey": "123",
		"numberKey": 1.0 / 3,
		"objectKey": map[string]interface{}{"foo": "bar"},
	}

	err := CoerceParams(schema, data)
	assert.NoError(t, err)
	assert.Equal(t, "123", data["stringkey"])
	assert.Equal(t, 1.0/3, data["numberKey"])
	assert.Equal(t, "bar", data["objectKey"].(map[string]interface{})["foo"])
}

func TestCoerceParams_Recursion(t *testing.T) {
	schema := &spec.Schema{Properties: map[string]*spec.Schema{
		"mapkey": {Properties: map[string]*spec.Schema{
			"intkey": {Type: integerType},
		}},
	}}
	data := map[string]interface{}{
		"mapkey": map[string]interface{}{
			"intkey": "123",
		},
	}

	err := CoerceParams(schema, data)
	assert.NoError(t, err)
	assert.Equal(t, 123, data["mapkey"].(map[string]interface{})["intkey"])
}
