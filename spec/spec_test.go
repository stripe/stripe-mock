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

func TestUnmarshal_UnsupportedField(t *testing.T) {
	// We don't support 'const'
	data := []byte(`{const: "hello"}`)
	var schema Schema
	err := json.Unmarshal(data, &schema)
	assert.Error(t, err)
}
