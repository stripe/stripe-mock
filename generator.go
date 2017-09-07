package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stripe/stripe-mock/spec"
	"github.com/stripe/stripe-mock/util"
)

var errExpansionNotSupported = fmt.Errorf("Expansion not supported")
var errNotSupported = fmt.Errorf("Expected response to be a list or include $ref")

// DataGenerator generates fixture response data based off a response schema, a
// set of definitions, and a fixture store.
type DataGenerator struct {
	definitions map[string]*spec.Schema
	fixtures    *spec.Fixtures
}

// Generate generates a fixture response.
func (g *DataGenerator) Generate(schema *spec.Schema, requestPath string, expansions *ExpansionLevel) (interface{}, error) {
	return g.generateInternal(schema, requestPath, expansions, nil)
}

func (g *DataGenerator) generateInternal(schema *spec.Schema, requestPath string, expansions *ExpansionLevel, existingData interface{}) (interface{}, error) {
	schema, err := g.maybeDereference(schema)
	if err != nil {
		return nil, err
	}

	// Determine if the requested expansions are possible
	if expansions != nil {
		for key := range expansions.expansions {
			if sort.SearchStrings(schema.XExpandableFields, key) ==
				len(schema.XExpandableFields) {
				return nil, errExpansionNotSupported
			}
		}
	}

	data, err := g.generateResource(schema)
	if err != nil {
		return nil, err
	}

	if schema.Properties != nil {
		listData, err := g.maybeGenerateList(
			schema.Properties, existingData, requestPath, expansions)
		if err != nil {
			return nil, err
		}
		if listData != nil {
			return listData, nil
		}

		for key, property := range schema.Properties {
			dataMap := data.(map[string]interface{})

			subSchema := property

			var subExpansions *ExpansionLevel
			if expansions != nil {
				var ok bool
				subExpansions, ok = expansions.expansions[key]

				var expansion *spec.Schema
				if property.XExpansionResources != nil {
					expansion = property.XExpansionResources.OneOf[0]
				}

				// Point to the expanded schema in either the case that an
				// expansion was requested on this field or we have a wildcard
				// expansion active.
				if expansion != nil && (ok || expansions.wildcard) {
					subSchema = expansion
				}
			}

			keyData, err := g.generateInternal(
				subSchema, requestPath, subExpansions, dataMap[key])
			if err == errNotSupported {
				continue
			}
			if err != nil {
				return nil, err
			}
			dataMap[key] = keyData
		}
	}

	return data, nil
}

func (g *DataGenerator) generateResource(schema *spec.Schema) (interface{}, error) {
	if schema.XResourceID == "" {
		if schema.Type == "" || schema.Type == "object" {
			return map[string]interface{}{}, nil
		} else {
			return nil, errNotSupported
		}
	}

	fixture, ok := g.fixtures.Resources[spec.ResourceID(schema.XResourceID)]
	if !ok {
		util.Warningf("No fixture for: %s", spec.ResourceID(schema.XResourceID))
		return map[string]interface{}{}, nil
	}

	return duplicateMap(fixture.(map[string]interface{})), nil
}

func (g *DataGenerator) maybeDereference(schema *spec.Schema) (*spec.Schema, error) {
	if schema.Ref != "" {
		definition, err := definitionFromJSONPointer(schema.Ref)
		if err != nil {
			return nil, err
		}

		newSchema, ok := g.definitions[definition]
		if !ok {
			return nil, fmt.Errorf("Couldn't dereference: %v", schema.Ref)
		}
		schema = newSchema
	}
	return schema, nil
}

func (g *DataGenerator) maybeGenerateList(properties map[string]*spec.Schema, existingData interface{}, requestPath string, expansions *ExpansionLevel) (interface{}, error) {
	object, ok := properties["object"]
	if !ok {
		return nil, nil
	}

	if object.Enum == nil {
		return nil, nil
	}

	if object.Enum[0] != "list" {
		return nil, nil
	}

	data, ok := properties["data"]
	if !ok {
		return nil, nil
	}

	if data.Items == nil {
		return nil, nil
	}

	itemsSchema, err := g.maybeDereference(data.Items)
	if err != nil {
		return nil, err
	}

	itemsData, err := g.generateInternal(itemsSchema, requestPath, expansions, nil)
	if err != nil {
		return nil, err
	}

	// This is written to hopefully be a little more forward compatible in that
	// it respects the list properties dictated by the included schema rather
	// than assuming its own.
	listData := make(map[string]interface{})
	for key := range properties {
		var val interface{}
		switch key {
		case "data":
			val = []interface{}{itemsData}
		case "has_more":
			val = false
		case "object":
			val = "list"
		case "total_count":
			val = 1
		case "url":
			existingDataMap, ok := existingData.(map[string]interface{})
			if ok {
				// Reuse a URL that came from fixture data if one is available
				val = existingDataMap["url"]
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

// duplicateArr is a helper method for duplicateMap.
func duplicateArr(dataArr []interface{}) []interface{} {
	copyArr := make([]interface{}, len(dataArr))

	for i, v := range dataArr {
		vMap, ok := v.(map[string]interface{})
		if ok {
			copyArr[i] = duplicateMap(vMap)
			continue
		}

		vArr, ok := v.([]interface{})
		if ok {
			copyArr[i] = duplicateArr(vArr)
			continue
		}

		copyArr[i] = v
	}

	return copyArr
}

// duplicateMap is a hacky way around the fact that there's no way to copy
// something like a map in Go. We need to copy a fixture so that we can modify
// and return it, which is why this exists.
func duplicateMap(dataMap map[string]interface{}) map[string]interface{} {
	copyMap := make(map[string]interface{})

	for k, v := range dataMap {
		vMap, ok := v.(map[string]interface{})
		if ok {
			copyMap[k] = duplicateMap(vMap)
			continue
		}

		vArr, ok := v.([]interface{})
		if ok {
			copyMap[k] = duplicateArr(vArr)
			continue
		}

		copyMap[k] = v
	}

	return copyMap
}

// definitionFromJSONPointer extracts the name of a JSON schema definition from
// a JSON pointer, so "#/components/schemas/charge" would become just "charge".
// This is a simplified workaround to avoid bringing in JSON schema
// infrastructure because we can guarantee that the spec we're producing will
// take a certain shape. If this gets too hacky, it will be better to put a more
// legitimate JSON schema parser in place.
func definitionFromJSONPointer(pointer string) (string, error) {
	parts := strings.Split(pointer, "/")

	if len(parts) != 4 ||
			parts[0] != "#" ||
			parts[1] != "components" ||
			parts[2] != "schemas" {
		return "", fmt.Errorf("Expected '#/components/schemas/...' but got '%v'", pointer)
	}

	return parts[3], nil
}
