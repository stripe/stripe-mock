package main

import (
	"fmt"
	"strings"
)

var notSupportedErr = fmt.Errorf("Expected response to be a list or include $ref")

type DataGenerator struct {
	definitions map[string]JSONSchema
	fixtures    *Fixtures
}

func (g *DataGenerator) maybeDereference(schema JSONSchema) (JSONSchema, error) {
	ref, ok := schema["$ref"].(string)
	if ok {
		definition, err := definitionFromJSONPointer(ref)
		if err != nil {
			return nil, err
		}

		schema, ok = g.definitions[definition]
		if !ok {
			return nil, fmt.Errorf("Couldn't dereference: %v", ref)
		}
	}
	return schema, nil
}

func (g *DataGenerator) generateResource(schema JSONSchema) (interface{}, error) {
	xResourceID, ok := schema["x-resourceId"].(string)
	if !ok {
		schemaType, ok := schema["type"].(string)
		if ok {
			if schemaType == "object" {
				return map[string]interface{}{}, nil
			}
			return nil, notSupportedErr
		}

		// Types are also allowed to be an array of types
		schemaTypes, ok := schema["type"].([]string)
		if ok {
			for _, schemaType := range schemaTypes {
				if schemaType == "object" {
					return map[string]interface{}{}, nil
				}
			}
			return nil, notSupportedErr
		}

		// Support schemas with no type annotation at all
		return map[string]interface{}{}, nil
	}

	fixture, ok := g.fixtures.Resources[ResourceID(xResourceID)]
	if !ok {
		return nil, fmt.Errorf("Expected fixtures to include %v", xResourceID)
	}
	return fixture, nil
}

func (g *DataGenerator) Generate(schema JSONSchema, requestPath string) (interface{}, error) {
	schema, err := g.maybeDereference(schema)
	if err != nil {
		return nil, err
	}

	data, err := g.generateResource(schema)
	if err != nil {
		return nil, err
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if ok {
		listData, err := g.maybeGenerateList(properties, requestPath)
		if err != nil {
			return nil, err
		}
		if listData != nil {
			return listData, nil
		}

		for key, property := range properties {
			keyData, err := g.Generate(property.(JSONSchema), requestPath)
			if err == notSupportedErr {
				continue
			}
			if err != nil {
				return nil, err
			}
			data.(map[string]interface{})[key] = keyData
		}
	}

	return data, nil
}

func (g *DataGenerator) maybeGenerateList(properties map[string]interface{}, requestPath string) (interface{}, error) {
	object, ok := properties["object"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	objectEnum, ok := object["enum"].([]interface{})
	if !ok {
		return nil, nil
	}

	if objectEnum[0] != interface{}("list") {
		return nil, nil
	}

	data, ok := properties["data"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	items, ok := data["items"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	itemsSchema, err := g.maybeDereference(items)
	if err != nil {
		return nil, err
	}

	itemsData, err := g.Generate(itemsSchema, requestPath)
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
			val = requestPath
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
