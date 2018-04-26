package coercer

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/stripe/stripe-mock/spec"
)

// CoerceParams coerces the types of certain parameters according to typing
// information from their corresponding JSON schema. This is useful because an
// input format like form-encoding doesn't support anything but strings, and
// we'd like to work with a slightly wider variety of types like booleans and
// integers.
func CoerceParams(schema *spec.Schema, data map[string]interface{}) error {
	for key, subSchema := range schema.Properties {
		val, ok := data[key]
		if !ok {
			continue
		}

		valMap, ok := val.(map[string]interface{})
		if ok {
			CoerceParams(subSchema, valMap)

			if subSchema.Type == arrayType {
				valSlice, err := parseIntegerIndexedMap(valMap)
				if err != nil {
					return err
				}

				if valSlice != nil {
					data[key] = valSlice
					val = valSlice
				}
			}

			// May fall through to the next segment where we iterate an array
			// and coerce it
		}

		valArr, ok := val.([]interface{})
		if ok {
			if subSchema.Items != nil {
				for i, itemVal := range valArr {
					itemValMap, ok := itemVal.(map[string]interface{})
					if ok {
						// Handles the case of an array of generic objects
						CoerceParams(subSchema.Items, itemValMap)
					} else if subSchema.Items.Type != "" {
						// Handles the case of an array of primitive types
						itemValCoerced, ok := coerceSchema(itemVal, subSchema.Items)
						if ok {
							valArr[i] = itemValCoerced
						}
					}
				}
			}

			continue
		}

		valCoerced, ok := coerceSchema(val, subSchema)
		if ok {
			data[key] = valCoerced
		}
	}

	return nil
}

//
// ---
//

// Various identifiers for types in JSON schema.
const (
	arrayType   = "array"
	booleanType = "boolean"
	integerType = "integer"
	numberType  = "number"
	objectType  = "object"
)

// maxSliceSize defines a somewhat arbitrary maximum size on an incoming
// integer-indexed map that we're willing to parse so that we don't run out of
// memory trying to allocate a slice.
const maxSliceSize = 1000

// numberPattern simply checks to see if an input string looks like a number.
var numberPattern = regexp.MustCompile(`\A\d+\z`)

// coercePrimitiveType tries to coerce a primitive type (e.g. bool, int, etc.)
// from the given generic interface{} value. On success it returns a coerced
// value with a boolean true. On failure (say the value wasn't a type that
// could be coerced) it returns nil and a boolean false.
func coercePrimitiveType(val interface{}, primitiveType string) (interface{}, bool) {
	valStr, ok := val.(string)
	if !ok {
		return nil, false
	}

	switch {
	case primitiveType == booleanType:
		valBool, err := strconv.ParseBool(valStr)
		if err != nil {
			valBool = false
		}
		return valBool, true

	case primitiveType == integerType:
		valInt, err := strconv.Atoi(valStr)
		if err != nil {
			valInt = 0
		}
		return valInt, true

	case primitiveType == numberType:
		valFloat, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			valFloat = 0.0
		}
		return valFloat, true
	}

	return nil, false
}

// coerceSchema tries to coerce a schema containing a primitive type from the
// given generic interface{} value.
//
// It's similar to coercePrimitiveType above (and indeed calls into it), but
// also handles the case of an anyOf schema that supports a number of different
// primitve types.
func coerceSchema(val interface{}, schema *spec.Schema) (interface{}, bool) {
	if isSchemaPrimitiveType(schema) {
		return coercePrimitiveType(val, schema.Type)
	} else if schema.AnyOf != nil {
		for _, subSchema := range schema.AnyOf {
			val, ok := coerceSchema(val, subSchema)
			if ok {
				return val, ok
			}
		}
	}

	return nil, false
}

// isSchemaPrimitiveType checks whether the given schema is a coercable
// primitive type (as opposed to an object or array).
//
// The conditional ladder in this function should be *identical* to the one in
// coercePrimitiveType (i.e., if support is added for a new type, it needs to
// be added in both places).
func isSchemaPrimitiveType(schema *spec.Schema) bool {
	if schema.Type == booleanType {
		return true
	}

	if schema.Type == integerType {
		return true
	}

	if schema.Type == numberType {
		return true
	}

	return false
}

// parseIntegerIndexedMap tries to parse a map that has all integer-indexed
// keys (e.g. { "0": ..., "1": "...", "2": "..." }) as a slice. We only try to
// do this when we know that the target schema requires an array.
func parseIntegerIndexedMap(valMap map[string]interface{}) ([]interface{}, error) {
	allNumberedIndexes := true
	biggestIndex := 0

	for index := range valMap {
		matched := numberPattern.MatchString(index)
		if !matched {
			allNumberedIndexes = false
			break
		}

		valInt, err := strconv.Atoi(index)
		if err != nil {
			allNumberedIndexes = false
			break
		}

		if valInt > biggestIndex {
			biggestIndex = valInt
		}
	}

	if !allNumberedIndexes {
		return nil, nil
	}

	if biggestIndex > maxSliceSize {
		return nil, fmt.Errorf("Index %v is too large, won't parse as slice", biggestIndex)
	}

	valSlice := make([]interface{}, biggestIndex+1)

	for index, val := range valMap {
		// Already checked error above
		indexInt, _ := strconv.Atoi(index)
		valSlice[indexInt] = val
	}

	return valSlice, nil
}
