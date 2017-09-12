package coercer

import (
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/stripe/stripe-mock/spec"
)

func TestCoerceParams_BooleanCoercion(t *testing.T) {
	schema := &spec.JSONSchema{Properties: map[string]*spec.JSONSchema{
		"boolkey": {Type: []string{booleanType}},
	}}
	data := map[string]interface{}{
		"boolkey": "true",
	}

	err := CoerceParams(schema, data)
	assert.NoError(t, err)
	assert.Equal(t, true, data["boolkey"])
}

func TestCoerceParams_IntegerCoercion(t *testing.T) {
	schema := &spec.JSONSchema{Properties: map[string]*spec.JSONSchema{
		"intkey": {Type: []string{integerType}},
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
		schema := &spec.JSONSchema{Properties: map[string]*spec.JSONSchema{
			"arraykey": {Type: []string{arrayType}},
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
		schema := &spec.JSONSchema{Properties: map[string]*spec.JSONSchema{
			"arraykey": {Type: []string{arrayType}},
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
		schema := &spec.JSONSchema{Properties: map[string]*spec.JSONSchema{
			"arraykey": {Type: []string{arrayType}},
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
		schema := &spec.JSONSchema{Properties: map[string]*spec.JSONSchema{
			"arraykey": {Type: []string{arrayType}},
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
	schema := &spec.JSONSchema{Properties: map[string]*spec.JSONSchema{
		"numberkey": {Type: []string{numberType}},
	}}
	data := map[string]interface{}{
		"numberkey": "123.45",
	}

	err := CoerceParams(schema, data)
	assert.NoError(t, err)
	assert.Equal(t, 123.45, data["numberkey"])
}

func TestCoerceParams_Recursion(t *testing.T) {
	schema := &spec.JSONSchema{Properties: map[string]*spec.JSONSchema{
		"mapkey": {Properties: map[string]*spec.JSONSchema{
			"intkey": {Type: []string{integerType}},
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
