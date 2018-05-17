package datareplacer

import (
	"testing"

	assert "github.com/stretchr/testify/require"
)

//
// Tests
//

func TestReplaceData_Basic(t *testing.T) {
	responseData := map[string]interface{}{
		"foo": "response-value",
	}

	ReplaceData(map[string]interface{}{
		"foo": "request-value",
	}, responseData)

	assert.Equal(t, map[string]interface{}{
		"foo": "request-value",
	}, responseData)
}

// Arrays are currently just replaced wholesale.
func TestReplaceData_Array(t *testing.T) {
	responseData := map[string]interface{}{
		"arr": []string{
			"response-value",
		},
	}

	ReplaceData(map[string]interface{}{
		"arr": []string{
			"request-value",
		},
	}, responseData)

	assert.Equal(t, map[string]interface{}{
		"arr": []string{
			"request-value",
		},
	}, responseData)
}

func TestReplaceData_Recursive(t *testing.T) {
	responseData := map[string]interface{}{
		"map": map[string]interface{}{
			"nested": "response-value",
		},
	}

	ReplaceData(map[string]interface{}{
		"map": map[string]interface{}{
			"nested": "request-value",
		},
	}, responseData)

	assert.Equal(t, map[string]interface{}{
		"map": map[string]interface{}{
			"nested": "request-value",
		},
	}, responseData)
}

func TestReplaceData_DontReplaceOnDifferentFields(t *testing.T) {
	responseData := map[string]interface{}{
		"other": "other-value",
	}

	ReplaceData(map[string]interface{}{
		"foo": "request-value",
	}, responseData)

	assert.Equal(t, map[string]interface{}{
		"other": "other-value",
	}, responseData)
}

func TestReplaceData_DontReplaceOnDifferentTypes(t *testing.T) {
	responseData := map[string]interface{}{
		"foo": "response-value",
	}

	ReplaceData(map[string]interface{}{
		"foo": 7,
	}, responseData)

	assert.Equal(t, map[string]interface{}{
		"foo": "response-value",
	}, responseData)
}
