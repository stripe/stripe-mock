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

	CoerceParams(schema, data)
	assert.Equal(t, true, data["boolkey"])
}

func TestCoerceParams_IntegerCoercion(t *testing.T) {
	schema := &spec.JSONSchema{Properties: map[string]*spec.JSONSchema{
		"intkey": {Type: []string{integerType}},
	}}
	data := map[string]interface{}{
		"intkey": "123",
	}

	CoerceParams(schema, data)
	assert.Equal(t, 123, data["intkey"])
}

func TestCoerceParams_NumberCoercion(t *testing.T) {
	schema := &spec.JSONSchema{Properties: map[string]*spec.JSONSchema{
		"numberkey": {Type: []string{numberType}},
	}}
	data := map[string]interface{}{
		"numberkey": "123.45",
	}

	CoerceParams(schema, data)
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

	CoerceParams(schema, data)
	assert.Equal(t, 123, data["mapkey"].(map[string]interface{})["intkey"])
}
