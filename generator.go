package main

import (
	"fmt"
	"strings"
)

var notSupportedErr = fmt.Errorf("Expected response to be a list or include $ref")

type DataGenerator struct {
	definitions map[string]*JSONSchema
	fixtures    *Fixtures
}

func (g *DataGenerator) maybeDereference(schema *JSONSchema) (*JSONSchema, error) {
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

func (g *DataGenerator) generateResource(schema *JSONSchema) (interface{}, error) {
	if schema.XResourceID == "" {
		// Technically type can also be just a string, but we're not going to
		// support this for now.
		if schema.Type != nil {
			for _, schemaType := range schema.Type {
				if schemaType == "object" {
					return map[string]interface{}{}, nil
				}
			}
			return nil, notSupportedErr
		}

		// Support schemas with no type annotation at all
		return map[string]interface{}{}, nil
	}

	fixture, ok := g.fixtures.Resources[ResourceID(schema.XResourceID)]
	if !ok {
		return map[string]interface{}{}, nil
	}

	return fixture, nil
}

func (g *DataGenerator) Generate(schema *JSONSchema, requestPath string, expansions *ExpansionLevel) (interface{}, error) {
	return g.generateInternal(schema, requestPath, expansions, nil)
}

func (g *DataGenerator) generateInternal(schema *JSONSchema, requestPath string, expansions *ExpansionLevel, existingData interface{}) (interface{}, error) {
	schema, err := g.maybeDereference(schema)
	if err != nil {
		return nil, err
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
			keyData, err := g.generateInternal(property, requestPath, expansions, dataMap[key])
			if err == notSupportedErr {
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

func (g *DataGenerator) maybeGenerateList(properties map[string]*JSONSchema, existingData interface{}, requestPath string, expansions *ExpansionLevel) (interface{}, error) {
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

// definitionFromJSONPointer extracts the name of a JSON schema definition from
// a JSON pointer, so "#/definitions/charge" would become just "charge". This
// is a simplified workaround to avoid bringing in JSON schema infrastructure
// because we can guarantee that the spec we're producing will take a certain
// shape. If this gets too hacky, it will be better to put a more legitimate
// JSON schema parser in place.
func definitionFromJSONPointer(pointer string) (string, error) {
	parts := strings.Split(pointer, "/")

	if parts[0] != "#" {
		return "", fmt.Errorf("Expected '#' in 0th part of pointer %v", pointer)
	}

	if parts[1] != "definitions" {
		return "", fmt.Errorf("Expected 'definitions' in 1st part of pointer %v",
			pointer)
	}

	if len(parts) > 3 {
		return "", fmt.Errorf("Pointer too long to be handle %v", pointer)
	}

	return parts[2], nil
}
