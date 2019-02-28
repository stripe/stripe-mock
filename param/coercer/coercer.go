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
		coercedVal, ok, err := coerceSubSchema(val, subSchema)
		if err != nil {
			return err
		}
		if ok {
			data[key] = coercedVal
		}
	}

	return nil
}

// coerceSubSchema coerces a sub-param value according to an arbitrary JSON sub-schema.
// It is named with "sub-schema" because it can be array, primitive, or object, unlike the main
// `CoerceParams` only expecting object type with properties. This is also used in coercing each
// sub-schema of anyOf or array.
func coerceSubSchema(val interface{}, subSchema *spec.Schema) (interface{}, bool, error) {
	if len(subSchema.Properties) == 0 {
		// Non-object schemas are anyOf, array, and primitive schemas.
		// Implicitly treats actual object schema with empty properties as non-object.
		return coerceNonObjectSchema(val, subSchema)
	}

	// `object` schema with properties
	valMap, ok := val.(map[string]interface{})
	var err error
	if ok {
		// unwrapping sub-schemas, and coerce contents in the map
		err = CoerceParams(subSchema, valMap)
	}
	return valMap, ok, err
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
	stringType  = "string"
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
			return nil, false
		}
		return valBool, true

	case primitiveType == integerType:
		valInt, err := strconv.Atoi(valStr)
		if err != nil {
			return nil, false
		}
		return valInt, true

	case primitiveType == numberType:
		valFloat, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return nil, false
		}
		return valFloat, true

	case primitiveType == stringType:
		return valStr, true
	}

	return nil, false
}

// coerceNonObjectSchema tries to coerce a non-object schema given generic interface{} value.
//
// It's similar to coercePrimitiveType above (and indeed calls into it), but
// also handles array and anyOf schema (supporting a number of different primitive types)
func coerceNonObjectSchema(val interface{}, schema *spec.Schema) (interface{}, bool, error) {
	if isSchemaPrimitiveType(schema) {
		if schema.Enum != nil {
			// assuming enum value isn't numeric string. when given anyOf schema with enum and
			// number, the numeric string won't falsely be taken as enum and miss its coercion
			_, isNumeric := coercePrimitiveType(val, numberType)
			if isNumeric {
				return nil, false, nil
			}
		}

		val, ok := coercePrimitiveType(val, schema.Type)
		return val, ok, nil
	}

	if schema.AnyOf != nil {
		for _, subSchema := range schema.AnyOf {
			val, ok, err := coerceSubSchema(val, subSchema)
			if ok {
				return val, ok, err
			}
		}
	}

	if schema.Type == arrayType {
		valMap, ok := val.(map[string]interface{})
		if ok {
			valSlice, err := parseIntegerIndexedMap(valMap)
			if err != nil {
				return nil, false, err
			}
			if valSlice != nil {
				val = valSlice
			}
		}

		valArr, ok := val.([]interface{})
		if schema.Items == nil {
			// underspecified array of primitive
			return val, ok, nil
		}

		if ok {
			allOk := true
			for i, itemVal := range valArr {
				itemValCoerced, itemOk, err := coerceSubSchema(itemVal, schema.Items)
				if err != nil {
					return nil, false, err
				}
				if itemOk {
					valArr[i] = itemValCoerced
				}
				skipNilItem := itemVal == nil
				allOk = allOk && (itemOk || skipNilItem)
			}
			if allOk {
				return valArr, true, nil
			}
		}
	}

	return nil, false, nil
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

	if schema.Type == stringType {
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
