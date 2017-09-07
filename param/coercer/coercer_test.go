package coercer

import (
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/stripe/stripe-mock/spec"
)

func TestCoerceParams_BooleanCoercion(t *testing.T) {
	schema := &spec.Schema{Properties: map[string]*spec.Schema{
		"boolkey": {Type: booleanType},
	}}
	data := map[string]interface{}{
		"boolkey": "true",
	}

	CoerceParams(schema, data)
	assert.Equal(t, true, data["boolkey"])
}

func TestCoerceParams_IntegerCoercion(t *testing.T) {
	schema := &spec.Schema{Properties: map[string]*spec.Schema{
		"intkey": {Type: integerType},
	}}
	data := map[string]interface{}{
		"intkey": "123",
	}

	CoerceParams(schema, data)
	assert.Equal(t, 123, data["intkey"])
}

func TestCoerceParams_NumberCoercion(t *testing.T) {
	schema := &spec.Schema{Properties: map[string]*spec.Schema{
		"numberkey": {Type: numberType},
	}}
	data := map[string]interface{}{
		"numberkey": "123.45",
	}

	CoerceParams(schema, data)
	assert.Equal(t, 123.45, data["numberkey"])
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

	CoerceParams(schema, data)
	assert.Equal(t, 123, data["mapkey"].(map[string]interface{})["intkey"])
}
