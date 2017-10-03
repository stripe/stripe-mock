package main

import (
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestStringifyKeysMapValue(t *testing.T) {
	assert.Equal(t,
		map[string]interface{}{
			"arrkey": []interface{}{
				123,
				map[string]interface{}{
					"intkey": 123,
				},
			},
			"boolkey": true,
			"intkey":  123,
			"mapkey": map[string]interface{}{
				"intkey": 123,
			},
		},
		stringifyKeysMapValue(map[interface{}]interface{}{
			"arrkey": []interface{}{
				123,
				map[interface{}]interface{}{
					"intkey": 123,
				},
			},
			"boolkey": true,
			"intkey":  123,
			"mapkey": map[interface{}]interface{}{
				"intkey": 123,
			},
		}),
	)
}
