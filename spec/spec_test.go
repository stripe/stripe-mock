package spec

import (
	"encoding/json"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestUnmarshal_Simple(t *testing.T) {
	data := []byte(`{"type": "string"}`)
	var schema Schema
	err := json.Unmarshal(data, &schema)
	assert.NoError(t, err)
	assert.Equal(t, "string", schema.Type)
}

func TestUnmarshal_ObjectWithAdditionalProperties(t *testing.T) {
	data := []byte(`{
		"type": "object",
		"additionalProperties": {
			"properties": {
				"prop": {
					"type": "number"
				}
			},
			"type": "object"
		}
	}`)
	var schema Schema
	err := json.Unmarshal(data, &schema)
	assert.NoError(t, err)
	assert.Equal(t, "object", schema.Type)
	assert.True(t, schema.AdditionalPropertiesAllowed)
	assert.Equal(t, "object", schema.AdditionalProperties.Type)
	assert.Equal(t, 1, len(schema.AdditionalProperties.Properties))
}

func TestUnmarshal_ObjectWithAdditionalPropertiesFalse(t *testing.T) {
	data := []byte(`{
		"type": "object",
		"additionalProperties": false
	}`)
	var schema Schema
	err := json.Unmarshal(data, &schema)
	assert.NoError(t, err)
	assert.Equal(t, "object", schema.Type)
	assert.False(t, schema.AdditionalPropertiesAllowed)
	assert.Nil(t, schema.AdditionalProperties)
}

func TestUnmarshal_ObjectWithAdditionalPropertiesDefault(t *testing.T) {
	data := []byte(`{
		"type": "object"
	}`)
	var schema Schema
	err := json.Unmarshal(data, &schema)
	assert.NoError(t, err)
	assert.Equal(t, "object", schema.Type)
	assert.True(t, schema.AdditionalPropertiesAllowed)
	assert.Nil(t, schema.AdditionalProperties)
}

func TestUnmarshal_UnsupportedField(t *testing.T) {
	// We don't support 'const'
	data := []byte(`{const: "hello"}`)
	var schema Schema
	err := json.Unmarshal(data, &schema)
	assert.Error(t, err)
}
