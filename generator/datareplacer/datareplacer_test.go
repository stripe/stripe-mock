package datareplacer

import (
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/stripe/stripe-mock/spec"
)

//
// Tests
//

func TestReplaceData_AnyOf(t *testing.T) {
	replacer := DataReplacer{Schema: &spec.Schema{
		Properties: map[string]*spec.Schema{
			"foo": {
				AnyOf: []*spec.Schema{
					{
						Type: spec.TypeString,
					},
				},
			},
		},
	}}

	// Incoming matching value (string, which is part of `AnyOf`)
	{
		responseData := map[string]interface{}{
			"foo": "response",
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": "request",
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": "request",
		}, responseData)
	}

	// Incoming non-matching value
	{
		responseData := map[string]interface{}{
			"foo": "response",
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": 123,
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": "response",
		}, responseData)
	}
}

func TestReplaceData_Array(t *testing.T) {
	replacer := DataReplacer{Schema: &spec.Schema{
		Properties: map[string]*spec.Schema{
			"foo": {
				Items: &spec.Schema{
					Type: spec.TypeString,
				},
				Type: spec.TypeArray,
			},
		},
	}}

	// Incoming matching array
	{
		responseData := map[string]interface{}{
			"foo": []interface{}{"response"},
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": []interface{}{"request"},
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": []interface{}{"request"},
		}, responseData)
	}

	// Incoming non-array
	{
		responseData := map[string]interface{}{
			"foo": []interface{}{"response"},
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": "hello",
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": []interface{}{"response"},
		}, responseData)
	}

	// Incoming array, but wrong element type
	{
		responseData := map[string]interface{}{
			"foo": []interface{}{"response"},
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": []interface{}{123},
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": []interface{}{"response"},
		}, responseData)
	}

	// Incoming empty array is allowed to replace
	{
		responseData := map[string]interface{}{
			"foo": []interface{}{"response"},
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": []interface{}{},
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": []interface{}{},
		}, responseData)
	}
}

func TestReplaceData_Boolean(t *testing.T) {
	replacer := DataReplacer{Schema: &spec.Schema{
		Properties: map[string]*spec.Schema{
			"foo": {
				Type: spec.TypeBoolean,
			},
		},
	}}

	// Incoming boolean
	{
		responseData := map[string]interface{}{
			"foo": false,
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": true,
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": true,
		}, responseData)
	}

	// Incoming non-boolean
	{
		responseData := map[string]interface{}{
			"foo": false,
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": "hello",
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": false,
		}, responseData)
	}
}

func TestReplaceData_Integer(t *testing.T) {
	replacer := DataReplacer{Schema: &spec.Schema{
		Properties: map[string]*spec.Schema{
			"foo": {
				Type: spec.TypeInteger,
			},
		},
	}}

	// Incoming integer
	{
		responseData := map[string]interface{}{
			"foo": 123,
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": 456,
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": 456,
		}, responseData)
	}

	// Incoming non-integer
	{
		responseData := map[string]interface{}{
			"foo": 123,
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": "hello",
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": 123,
		}, responseData)
	}
}

func TestReplaceData_Number(t *testing.T) {
	replacer := DataReplacer{Schema: &spec.Schema{
		Properties: map[string]*spec.Schema{
			"foo": {
				Type: spec.TypeNumber,
			},
		},
	}}

	// Incoming number
	{
		responseData := map[string]interface{}{
			"foo": 1.23,
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": 4.56,
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": 4.56,
		}, responseData)
	}

	// Incoming non-number
	{
		responseData := map[string]interface{}{
			"foo": 1.23,
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": "hello",
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": 1.23,
		}, responseData)
	}
}

func TestReplaceData_String(t *testing.T) {
	replacer := DataReplacer{Schema: &spec.Schema{
		Properties: map[string]*spec.Schema{
			"foo": {
				Type: spec.TypeString,
			},
		},
	}}

	// Incoming string
	{
		responseData := map[string]interface{}{
			"foo": "response",
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": "request",
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": "request",
		}, responseData)
	}

	// Incoming non-string
	{
		responseData := map[string]interface{}{
			"foo": "response",
		}

		replacer.ReplaceData(map[string]interface{}{
			"foo": 123,
		}, responseData)

		assert.Equal(t, map[string]interface{}{
			"foo": "response",
		}, responseData)
	}
}

func TestReplaceData_Deferences(t *testing.T) {
	replacer := DataReplacer{
		Definitions: map[string]*spec.Schema{
			"foo_object": {
				Type: spec.TypeString,
			},
		},
		Schema: &spec.Schema{
			Properties: map[string]*spec.Schema{
				"foo": {
					Ref: "#/definitions/foo_object",
				},
			},
		},
	}

	responseData := map[string]interface{}{
		"bar": "response",
	}

	replacer.ReplaceData(map[string]interface{}{
		"bar": "request",
	}, responseData)

	assert.Equal(t, map[string]interface{}{
		"bar": "response",
	}, responseData)
}

func TestReplaceData_Nested(t *testing.T) {
	replacer := DataReplacer{Schema: &spec.Schema{
		Properties: map[string]*spec.Schema{
			"foo": {
				Properties: map[string]*spec.Schema{
					"bar": {
						Type: spec.TypeString,
					},
				},
			},
		},
	}}

	responseData := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": "response",
		},
	}

	replacer.ReplaceData(map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": "request",
		},
	}, responseData)

	assert.Equal(t, map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": "request",
		},
	}, responseData)
}

// Nothing gets replaced if we don't find the field in the schema.
func TestReplaceData_NotInSchema(t *testing.T) {
	replacer := DataReplacer{Schema: &spec.Schema{
		Properties: map[string]*spec.Schema{
			"foo": {
				Type: spec.TypeString,
			},
		},
	}}

	responseData := map[string]interface{}{
		"bar": "response",
	}

	replacer.ReplaceData(map[string]interface{}{
		"bar": "request",
	}, responseData)

	assert.Equal(t, map[string]interface{}{
		"bar": "response",
	}, responseData)
}
