package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/stripe/stripe-mock/spec"
)

var errExpansionNotSupported = fmt.Errorf("Expansion not supported")

// DataGenerator generates fixture response data based off a response schema, a
// set of definitions, and a fixture store.
type DataGenerator struct {
	definitions map[string]*spec.Schema
	fixtures    *spec.Fixtures
}

// Generate generates a fixture response.
func (g *DataGenerator) Generate(schema *spec.Schema, requestPath string, expansions *ExpansionLevel) (interface{}, error) {
	return g.generateInternal(schema, requestPath, expansions, nil, fmt.Sprintf("Responding to %s:\n", requestPath))
}

// The purpose of this simple wrapper is so that we can write "example = nil"
// to indicate that no example is provided, as distinct from
// "example = &Example{value: nil}" for an example which is a null value.
type Example struct {
	value interface{}
}

func (g *DataGenerator) generateInternal(schema *spec.Schema, requestPath string, expansions *ExpansionLevel, example *Example, context string) (interface{}, error) {
	// This is a bit of a mess. We don't have an elegant fully-general approach to
	// generating examples, just a bunch of specific cases that we know how to
	// handle. If we find ourselves in a situation that doesn't match any of the
	// cases, then we fall through to the end of the function and panic().
	// Obviously this is fragile, so we have a unit test that makes sure it works
	// correctly on every resource; hopefully this will at least allow us to catch
	// any errors in advance.

	schema, context, err := g.maybeDereference(schema, context)
	if err != nil {
		return nil, err
	}

	// Determine if the requested expansions are possible
	if expansions != nil && schema.XExpandableFields != nil {
		for key := range expansions.expansions {
			if sort.SearchStrings(*schema.XExpandableFields, key) ==
				len(*schema.XExpandableFields) {
				return nil, errExpansionNotSupported
			}
		}
	}

	if (example == nil || example.value == nil) && schema.XResourceID != "" {
		// Use the fixture as our example. (Note that if the caller gave us a
		// non-trivial example, we prefer it instead, because it's probably more
		// relevant in context.)
		fixture, ok := g.fixtures.Resources[spec.ResourceID(schema.XResourceID)]
		if !ok {
			panic(fmt.Sprintf("%sMissing fixture for: %s", context, schema.XResourceID))
		}
		example = &Example{value: fixture}
		context = fmt.Sprintf("%sUsing fixture '%s':\n", context, schema.XResourceID)
	}

	if schema.XExpansionResources != nil {
		if expansions != nil {
			// We're expanding this specific object
			result, err := g.generateInternal(
				schema.XExpansionResources.OneOf[0], requestPath, expansions, nil,
				fmt.Sprintf("%sExpanding optional expandable field:\n", context))
			return result, err
		} else {
			// We're not expanding this specific object. Our example should be of the
			// unexpanded form, which is the first branch of the AnyOf
			result, err := g.generateInternal(
				schema.AnyOf[0], requestPath, expansions, example,
				fmt.Sprintf("%sNot expanding optional expandable field:\n", context))
			return result, err
		}
	}

	if len(schema.AnyOf) == 1 && schema.Nullable {
		if example != nil && example.value == nil {
			if expansions == nil {
				return nil, nil
			}
		} else {
			// Since there's only one subschema, we can confidently recurse into it
			result, err := g.generateInternal(
				schema.AnyOf[0], requestPath, expansions, example,
				fmt.Sprintf("%sChoosing only branch of anyOf:\n", context))
			return result, err
		}
	}

	if len(schema.AnyOf) != 0 {
		// Just generate an example of the first subschema. Note that we don't pass
		// in any example, even if we have an example available, because we don't
		// know which branch of the AnyOf the example corresponds to.
		result, err := g.generateInternal(
			schema.AnyOf[0], requestPath, expansions, nil,
			fmt.Sprintf("%sChoosing first branch of anyOf:\n", context))
		return result, err
	}

	if isListResource(schema) {
		// We special-case list resources and always fill in the list with at least
		// one item of data, regardless of what was present in the example
		listData, err := g.generateListResource(
			schema, requestPath, expansions, example, context)
		return listData, err
	}

	if example == nil {
		// If none of the above conditions met, we've run out of ways of generating
		// examples from scratch, so we can only raise an error.
		panic(fmt.Sprintf("%sCannot find or generate example for: %s", context, schema))
	}

	if example.value == nil {
		if expansions != nil {
			panic(fmt.Sprintf("%sWe were asked to expand a key, but our example "+
				"has null for that key.", context))
		}
		return nil, nil
	}

	if schema.Type == "boolean" || schema.Type == "integer" ||
		schema.Type == "number" || schema.Type == "string" {
		return example.value, nil
	}

	if schema.Type == "object" && schema.Properties == nil {
		// For a generic object type with no particular properties specified, we
		// assume it must not contain any expandable fields or list resources
		return example.value, nil
	}

	if schema.Type == "array" {
		// For lists that aren't contained in a list-object, we assume they do not
		// contain any expandable fields or list resources
		return example.value, nil
	}

	if schema.Type == "object" && schema.Properties != nil {
		exampleMap, ok := example.value.(map[string]interface{})
		if !ok {
			panic(fmt.Sprintf(
				"%sSchema is an object:\n%s\n\nBut example is (type: %v):\n%s",
				context, schema, reflect.TypeOf(example.value), example.value))
		}

		resultMap := make(map[string]interface{})

		for key, subSchema := range schema.Properties {

			var subExpansions *ExpansionLevel
			if expansions != nil {
				subExpansions = expansions.expansions[key]
				if subExpansions == nil && expansions.wildcard {
					// No expansion was provided for this key but the wildcard bit is set,
					// so make a fake expansion
					subExpansions = &ExpansionLevel{
						expansions: make(map[string]*ExpansionLevel),
						wildcard:   false,
					}
				}
			}

			var subExample *Example
			subExampleValue, exampleHasKey := exampleMap[key]
			if exampleHasKey {
				subExample = &Example{value: subExampleValue}
			}

			if !exampleHasKey && subExpansions == nil {
				// If the example omitted this key, then so do we; unless we were asked
				// to expand the key, in which case we'll have to generate an example
				// from scratch.
				continue
			}

			subValue, err := g.generateInternal(
				subSchema, requestPath, subExpansions, subExample,
				fmt.Sprintf("%sIn property '%s' of object:\n", context, key))
			if err != nil {
				return nil, err
			}
			resultMap[key] = subValue
		}

		return resultMap, nil
	}

	// If the schema is of the format we expect, this shouldn't ever happen.
	panic(fmt.Sprintf(
		"%sEncountered unusual scenario:\nschema=%s\nexample=%+v",
		context, schema, example))
}

func (g *DataGenerator) maybeDereference(schema *spec.Schema, context string) (*spec.Schema, string, error) {
	if schema.Ref != "" {
		definition := definitionFromJSONPointer(schema.Ref)

		newSchema, ok := g.definitions[definition]
		if !ok {
			panic(fmt.Sprintf("Couldn't dereference: %v", schema.Ref))
		}
		context = fmt.Sprintf("%sDereferencing '%s':\n", context, schema.Ref)
		schema = newSchema
	}
	return schema, context, nil
}

func (g *DataGenerator) generateListResource(schema *spec.Schema, requestPath string, expansions *ExpansionLevel, example *Example, context string) (interface{}, error) {

	var itemExpansions *ExpansionLevel
	if expansions != nil {
		itemExpansions = expansions.expansions["data"]
	}

	itemData, err := g.generateInternal(
		schema.Properties["data"].Items,
		requestPath,
		itemExpansions,
		nil,
		fmt.Sprintf("%sPopulating list resource:\n", context))
	if err != nil {
		return nil, err
	}

	// This is written to hopefully be a little more forward compatible in that
	// it respects the list properties dictated by the included schema rather
	// than assuming its own.
	listData := make(map[string]interface{})
	for key, subSchema := range schema.Properties {
		var val interface{}
		switch key {
		case "data":
			val = []interface{}{itemData}
		case "has_more":
			val = false
		case "object":
			val = "list"
		case "total_count":
			val = 1
		case "url":
			if len(subSchema.Enum) == 1 {
				val = subSchema.Enum[0]
			} else if example != nil {
				// If an example was provided, we can assume it has the correct format
				example := example.value.(map[string]interface{})
				val = example["url"]
			} else {
				val = requestPath
			}
		default:
			val = nil
		}
		listData[key] = val
	}
	return listData, nil
}

// ---

func isListResource(schema *spec.Schema) bool {
	if schema.Type != "object" || schema.Properties == nil {
		return false
	}

	object, ok := schema.Properties["object"]
	if !ok || object.Enum == nil || object.Enum[0] != "list" {
		return false
	}

	data, ok := schema.Properties["data"]
	if !ok || data.Items == nil {
		return false
	}

	return true
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
