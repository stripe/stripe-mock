package main

import (
	"fmt"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestGenerateResponseData(t *testing.T) {
	var data interface{}
	var err error
	var generator DataGenerator

	// basic reference
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(
		JSONSchema(map[string]interface{}{"$ref": "#/definitions/charge"}), "")

	assert.Nil(t, err)
	assert.Equal(t,
		testFixtures.Resources["charge"].(map[string]interface{})["id"],
		data.(map[string]interface{})["id"])

	// list
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(
		JSONSchema(map[string]interface{}{
			"properties": map[string]interface{}{
				"data": map[string]interface{}{
					"items": map[string]interface{}{
						"$ref": "#/definitions/charge",
					},
				},
				"has_more": nil,
				"object": map[string]interface{}{
					"enum": []interface{}{"list"},
				},
				"total_count": nil,
				"url":         nil,
			},
		}), "/v1/charges")
	assert.Nil(t, err)
	assert.Equal(t, "list", data.(map[string]interface{})["object"])
	assert.Equal(t, "/v1/charges", data.(map[string]interface{})["url"])
	assert.Equal(t,
		testFixtures.Resources["charge"].(map[string]interface{})["id"],
		data.(map[string]interface{})["data"].([]interface{})[0].(map[string]interface{})["id"])

	// error: unhandled JSON schema type
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(
		JSONSchema(map[string]interface{}{}), "")
	assert.Equal(t,
		fmt.Errorf("Expected response to be a list or include $ref"),
		err)

	// error: no definition in OpenAPI
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(
		JSONSchema(map[string]interface{}{"$ref": "#/definitions/doesnt-exist"}), "")
	assert.Equal(t,
		fmt.Errorf("Expected definitions to include doesnt-exist"),
		err)

	// error: no fixture
	generator = DataGenerator{
		testSpec.Definitions,
		// this is an empty set of fixtures
		&Fixtures{
			Resources: map[ResourceID]interface{}{},
		},
	}
	data, err = generator.Generate(
		JSONSchema(map[string]interface{}{"$ref": "#/definitions/charge"}), "")
	assert.Equal(t,
		fmt.Errorf("Expected fixtures to include charge"),
		err)
}

func TestDefinitionFromJSONPointer(t *testing.T) {
	definition, err := definitionFromJSONPointer("#/definitions/charge")
	assert.Nil(t, err)
	assert.Equal(t, "charge", definition)
}
