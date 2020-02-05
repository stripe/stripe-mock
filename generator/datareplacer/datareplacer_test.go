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

func TestReplaceData_Array(t *testing.T) {
	// We try to replace arrays wholesale.
	{
		responseData := map[string]interface{}{
			"arr": []interface{}{
				"response-value",
			},
		}

		ReplaceData(map[string]interface{}{
			"arr": []interface{}{
				"request-value",
			},
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"arr": []interface{}{
				"request-value",
			},
		}, responseData)
	}

	// But don't replace them if the types of their elements don't match.
	{
		responseData := map[string]interface{}{
			"arr": []interface{}{
				"response-value",
			},
		}

		ReplaceData(map[string]interface{}{
			"arr": []interface{}{
				7,
			},
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"arr": []interface{}{
				interface{}("response-value"),
			},
		}, responseData)
	}

	// Or if we don't have enough elements in either array to determine whether
	// they're supposed to the same type.
	{
		responseData := map[string]interface{}{
			"arr": []interface{}{},
		}

		ReplaceData(map[string]interface{}{
			"arr": []interface{}{
				"request-value",
			},
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"arr": []interface{}{},
		}, responseData)
	}
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
