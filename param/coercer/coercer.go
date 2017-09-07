package coercer

import (
	"strconv"

	"github.com/stripe/stripe-mock/spec"
)

// CoerceParams coerces the types of certain parameters according to typing
// information from their corresponding JSON schema. This is useful because an
// input format like form-encoding doesn't support anything but strings, and
// we'd like to work with a slightly wider variety of types like booleans and
// integers.
//
// Currently only the coercion of strings to bool and int64 is supported.
func CoerceParams(schema *spec.Schema, data map[string]interface{}) {
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
		if ok {
			switch {
			case subSchema.Type == booleanType:
				valBool, err := strconv.ParseBool(valStr)
				if err != nil {
					valBool = false
				}
				data[key] = valBool

			case subSchema.Type == integerType:
				valInt, err := strconv.Atoi(valStr)
				if err != nil {
					valInt = 0
				}
				data[key] = valInt

			case subSchema.Type == numberType:
				valFloat, err := strconv.ParseFloat(valStr, 64)
				if err != nil {
					valFloat = 0.0
				}
				data[key] = valFloat
			}
		}
	}
}

//
// ---
//

// booleanType is the name of the boolean type in a JSON schema.
const booleanType = "boolean"

// integerType is the name of the integer type in a JSON schema.
const integerType = "integer"

const numberType = "number"
