package main

import (
	"fmt"
	"strings"
)

var notSupportedErr = fmt.Errorf("Expected response to be a list or include $ref")

type DataGenerator struct {
	definitions map[string]OpenAPIDefinition
	fixtures    *Fixtures
}

func (g *DataGenerator) Generate(schema JSONSchema, requestPath string) (interface{}, error) {
	ref, ok := schema["$ref"].(string)
	if ok {
		return g.generateResource(ref)
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil, notSupportedErr
	}

	object, ok := properties["object"].(map[string]interface{})
	if !ok {
		return nil, notSupportedErr
	}

	objectEnum, ok := object["enum"].([]interface{})
	if !ok {
		return nil, notSupportedErr
	}

	if objectEnum[0] != interface{}("list") {
		return nil, notSupportedErr
	}

	data, ok := properties["data"].(map[string]interface{})
	if !ok {
		return nil, notSupportedErr
	}

	items, ok := data["items"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Expected list to include items schema")
	}

	itemsRef, ok := items["$ref"].(string)
	if !ok {
		return nil, fmt.Errorf("Expected items schema to include $ref")
	}

	innerData, err := g.generateResource(itemsRef)
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
			val = []interface{}{innerData}
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

func (g *DataGenerator) generateResource(pointer string) (interface{}, error) {
	definition, err := definitionFromJSONPointer(pointer)
	if err != nil {
		return nil, fmt.Errorf("Error extracting definition: %v", err)
	}

	resource, ok := g.definitions[definition]
	if !ok {
		return nil, fmt.Errorf("Expected definitions to include %v", definition)
	}

	fixture, ok := g.fixtures.Resources[resource.XResourceID]
	if !ok {
		return nil, fmt.Errorf("Expected fixtures to include %v", resource.XResourceID)
	}

	return fixture, nil
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
