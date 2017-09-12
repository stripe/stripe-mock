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
//
// Currently only the coercion of strings to bool and int64 is supported.
func CoerceParams(schema *spec.JSONSchema, data map[string]interface{}) error {
	for key, subSchema := range schema.Properties {
		val, ok := data[key]
		if !ok {
			continue
		}

		valMap, ok := val.(map[string]interface{})
		if ok {
			CoerceParams(subSchema, valMap)

			if hasType(subSchema, arrayType) && !hasType(subSchema, objectType) {
				valSlice, err := parseIntegerIndexedMap(valMap)
				if err != nil {
					return err
				}

				if valSlice != nil {
					data[key] = valSlice
				}
			}

			continue
		}

		valStr, ok := val.(string)
		if ok {
			switch {
			case hasType(subSchema, booleanType):
				valBool, err := strconv.ParseBool(valStr)
				if err != nil {
					valBool = false
				}
				data[key] = valBool

			case hasType(subSchema, integerType):
				valInt, err := strconv.Atoi(valStr)
				if err != nil {
					valInt = 0
				}
				data[key] = valInt

			case hasType(subSchema, numberType):
				valFloat, err := strconv.ParseFloat(valStr, 64)
				if err != nil {
					valFloat = 0.0
				}
				data[key] = valFloat
			}
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

func hasType(schema *spec.JSONSchema, targetTypeStr string) bool {
	for _, typeStr := range schema.Type {
		if typeStr == targetTypeStr {
			return true
		}
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
