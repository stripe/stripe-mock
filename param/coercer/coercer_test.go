package coercer

import (
	"testing"

	"github.com/brandur/stripestub/spec"
	assert "github.com/stretchr/testify/require"
)

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
