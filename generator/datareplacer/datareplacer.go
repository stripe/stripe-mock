package datareplacer

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/stripe/stripe-mock/spec"
)

// DataReplacer takes a generated response and replaces values in it that share
// a name and type of parameters that were sent in with the request, as
// determined by the associated OpenAPI schema and the types of incoming
// values.
//
// This is designed to have the effect of making returned fixtures more
// realistic while also staying a simple heuristic that doesn't require very
// much maintenance.
type DataReplacer struct {
	Definitions map[string]*spec.Schema
	Schema      *spec.Schema
}

// ReplaceData projects data from the incoming request into response data as
// appropriate.
func (r *DataReplacer) ReplaceData(requestData map[string]interface{}, responseData map[string]interface{}) map[string]interface{} {
	schema := r.Schema
	if schema != nil {
		schema, _ = r.maybeDereference(r.Schema, "")
	}

	return r.replaceDataInternal(requestData, responseData, schema)
}

// Identical to the above except that we pass a schema as argument so that we
// can easily have the relevant one during recursion.
func (r *DataReplacer) replaceDataInternal(requestData map[string]interface{}, responseData map[string]interface{}, schema *spec.Schema) map[string]interface{} {
	for k, requestValue := range requestData {
		responseValue, ok := responseData[k]

		// Recursively call in to replace data, but only if the key is
		// in both maps.
		//
		// A fairly obvious improvement here is if a key is in the
		// request but not present in the response, then check the
		// canonical schema to see if it's there. It might be an
		// optional field that doesn't appear in the fixture, and if it
		// was given to us with the request, we probably want to
		// include it.
		if ok {
			requestKeyMap, requestKeyOK := requestValue.(map[string]interface{})
			responseKeyMap, responseKeyOK := responseValue.(map[string]interface{})

			var kSchema *spec.Schema
			if schema != nil {
				kSchema = schema.Properties[k]

				if kSchema != nil {
					kSchema, _ = r.maybeDereference(kSchema, "")
				}
			}

			if requestKeyOK && responseKeyOK {
				responseData[k] = r.replaceDataInternal(requestKeyMap, responseKeyMap, kSchema)
			} else {
				// In the non-map case, just set the respons key's value to
				// what was in the request, but only if both values are the
				// same type (this is to prevent problems where a field is set
				// as an ID, but the response field is the hydrated object of
				// that).
				//
				// While this will largely be "good enough", there's some
				// obvious cases that aren't going to be handled correctly like
				// index-based array updates (e.g.,
				// `additional_owners[1][name]=...`). I'll have to iron out
				// that rough edges later on.
				if r.isSameType(kSchema, requestValue) {
					responseData[k] = requestValue
				}
			}
		}
	}

	return responseData
}

func (r *DataReplacer) isSameType(schema *spec.Schema, requestValue interface{}) bool {
	if schema == nil {
		return false
	}

	value := reflect.ValueOf(requestValue)

	// Reflect in Go has the concept of a "zero Value" (not be confused with a
	// type's zero value with a lowercase "v") and asking for Type on one will
	// panic. I'm not exactly sure under what conditions these are generated,
	// but they are occasionally, so here we hedge against them.
	//
	// https://github.com/stripe/stripe-mock/issues/75
	if !value.IsValid() {
		return false
	}

	valueType := value.Type()
	valueKind := valueType.Kind()

	switch {
	// In the case of `anyOf`, allow replacement if any of the schema branches apply.
	case len(schema.AnyOf) > 0:
		for _, anyOfSchema := range schema.AnyOf {
			anyOfSchema, _ := r.maybeDereference(anyOfSchema, "")
			if r.isSameType(anyOfSchema, requestValue) {
				return true
			}
		}

	case schema.Type == spec.TypeArray:
		valueSlice, ok := requestValue.([]interface{})

		// Incoming value is not an array
		if !ok {
			return false
		}

		// Allow the replacement if completely empty. In practice, this
		// should never happen because you can't send an empty array via
		// form data, but we'll cover the case anyway.
		if len(valueSlice) < 1 {
			return true
		}

		itemsSchema := schema.Items
		if itemsSchema == nil {
			return true
		}

		itemsSchema, _ = r.maybeDereference(itemsSchema, "")

		// Allow the replacement if the first item in the incoming slice is
		// compatible with the array's `items` schema.
		return r.isSameType(itemsSchema, valueSlice[0])

	case schema.Type == spec.TypeBoolean:
		return valueKind == reflect.Bool

	case schema.Type == spec.TypeInteger:
		return isIntegerKind(valueKind)

	case schema.Type == spec.TypeNumber:
		return isIntegerKind(valueKind) || valueKind == reflect.Float32 || valueKind == reflect.Float64

	// Don't try to replace objects for now, the likelihood is that they're
	// not compatible between request and response anyway.
	case schema.Type == spec.TypeObject:
		return false

	case schema.Type == spec.TypeString:
		return valueKind == reflect.String

	default:
		panic(fmt.Sprintf("Data replacer doesn't know how to handle schema: %+v", schema))
	}

	// Unreachable because of `default` above
	return false
}

func (r *DataReplacer) maybeDereference(schema *spec.Schema, context string) (*spec.Schema, string) {
	if schema.Ref != "" {
		definition := definitionFromJSONPointer(schema.Ref)

		newSchema, ok := r.Definitions[definition]
		if !ok {
			panic(fmt.Sprintf("Couldn't dereference: %v", schema.Ref))
		}
		context = fmt.Sprintf("%sDereferencing '%s':\n", context, schema.Ref)
		schema = newSchema
	}
	return schema, context
}

// definitionFromJSONPointer extracts the name of a JSON schema definition from
// a JSON pointer, so "#/components/schemas/charge" would become just "charge".
// This is a simplified workaround to avoid bringing in JSON schema
// infrastructure because we can guarantee that the spec we're producing will
// take a certain shape. If this gets too hacky, it will be better to put a more
// legitimate JSON schema parser in place.
func definitionFromJSONPointer(pointer string) string {
	parts := strings.Split(pointer, "/")

	if len(parts) != 4 ||
		parts[0] != "#" ||
		parts[1] != "components" ||
		parts[2] != "schemas" {
		panic(fmt.Sprintf("Expected '#/components/schemas/...' but got '%v'", pointer))
	}
	return parts[3]
}

func isIntegerKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int:
		return true
	case reflect.Int8:
		return true
	case reflect.Int16:
		return true
	case reflect.Int32:
		return true
	case reflect.Int64:
		return true

	case reflect.Uint:
		return true
	case reflect.Uint8:
		return true
	case reflect.Uint16:
		return true
	case reflect.Uint32:
		return true
	case reflect.Uint64:
		return true
	}

	return false
}
