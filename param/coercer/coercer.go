package coercer

import (
	"strconv"

	"github.com/brandur/stripestub/spec"
)

// CoerceParams coerces the types of certain parameters according to typing
// information from their corresponding JSON schema. This is useful because an
// input format like form-encoding doesn't support anything but strings, and
// we'd like to work with a slightly wider variety of types like integers.
//
// Currently only the coercion of strings to int64 is supported.
func CoerceParams(schema *spec.JSONSchema, data map[string]interface{}) {
	for key, subSchema := range schema.Properties {
		val, ok := data[key]
		if !ok {
			continue
		}

		valMap, ok := val.(map[string]interface{})
		if ok {
			CoerceParams(subSchema, valMap)
			continue
		}

		valStr, ok := val.(string)
		if ok && hasType(subSchema, integerType) {
			valInt, err := strconv.Atoi(valStr)
			if err != nil {
				valInt = 0
			}
			data[key] = valInt
		}
	}
}

//
// ---
//

// integerType is the name of the integer type in a JSON schema.
const integerType = "integer"

func hasType(schema *spec.JSONSchema, targetTypeStr string) bool {
	for _, typeStr := range schema.Type {
		if typeStr == targetTypeStr {
			return true
		}
	}
	return false
}
