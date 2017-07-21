package main

import (
	"fmt"
	"testing"

	"github.com/brandur/stripelocal/spec"
	assert "github.com/stretchr/testify/require"
)

var listSchema *spec.JSONSchema

func init() {
	listSchema = &spec.JSONSchema{
		Properties: map[string]*spec.JSONSchema{
			"data": {
				Items: &spec.JSONSchema{
					Ref: "#/definitions/charge",
				},
			},
			"has_more": nil,
			"object": {
				Enum: []string{"list"},
			},
			"total_count": nil,
			"url":         nil,
		},
	}
}

func TestGenerateResponseData(t *testing.T) {
	var data interface{}
	var err error
	var generator DataGenerator

	// basic reference
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(
		&spec.JSONSchema{Ref: "#/definitions/charge"}, "", nil)
	assert.Nil(t, err)
	assert.Equal(t,
		testFixtures.Resources["charge"].(map[string]interface{})["id"],
		data.(map[string]interface{})["id"])

	// Makes sure that customer is *not* expanded
	assert.Equal(t,
		testFixtures.Resources["customer"].(map[string]interface{})["id"],
		data.(map[string]interface{})["customer"])

	// expansion
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(
		&spec.JSONSchema{Ref: "#/definitions/charge"},
		"",
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{"customer": nil}})
	assert.Nil(t, err)
	assert.Equal(t,
		testFixtures.Resources["customer"].(map[string]interface{})["id"],
		data.(map[string]interface{})["customer"].(map[string]interface{})["id"])

	// bad expansion
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(
		&spec.JSONSchema{Ref: "#/definitions/charge"},
		"",
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{"id": nil}})
	assert.Equal(t, err, errExpansionNotSupported)

	// bad nested expansion
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(
		&spec.JSONSchema{Ref: "#/definitions/charge"},
		"",
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{"customer.id": nil}})
	assert.Equal(t, err, errExpansionNotSupported)

	// wildcard expansion
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(
		&spec.JSONSchema{Ref: "#/definitions/charge"},
		"",
		&ExpansionLevel{wildcard: true})
	assert.Nil(t, err)
	assert.Equal(t,
		testFixtures.Resources["customer"].(map[string]interface{})["id"],
		data.(map[string]interface{})["customer"].(map[string]interface{})["id"])

	// list
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(listSchema, "/v1/charges", nil)
	assert.Nil(t, err)
	assert.Equal(t, "list", data.(map[string]interface{})["object"])
	assert.Equal(t, "/v1/charges", data.(map[string]interface{})["url"])
	assert.Equal(t,
		testFixtures.Resources["charge"].(map[string]interface{})["id"],
		data.(map[string]interface{})["data"].([]interface{})[0].(map[string]interface{})["id"])

	// nested list
	generator = DataGenerator{
		testSpec.Definitions,
		&spec.Fixtures{
			Resources: map[spec.ResourceID]interface{}{
				spec.ResourceID("charge"): map[string]interface{}{"id": "ch_123"},
				spec.ResourceID("with_charges_list"): map[string]interface{}{
					"charges_list": map[string]interface{}{
						"url": "/v1/from_charges_list",
					},
				},
			},
		},
	}
	data, err = generator.Generate(
		&spec.JSONSchema{
			Properties: map[string]*spec.JSONSchema{
				"charges_list": listSchema,
			},
			XResourceID: "with_charges_list",
		}, "", nil)
	assert.Nil(t, err)
	chargesList := data.(map[string]interface{})["charges_list"]
	assert.Equal(t, "list", chargesList.(map[string]interface{})["object"])
	assert.Equal(t, "/v1/from_charges_list", chargesList.(map[string]interface{})["url"])
	assert.Equal(t,
		testFixtures.Resources["charge"].(map[string]interface{})["id"],
		chargesList.(map[string]interface{})["data"].([]interface{})[0].(map[string]interface{})["id"])

	// no fixture (returns an empty object)
	generator = DataGenerator{
		testSpec.Definitions,
		// this is an empty set of fixtures
		&spec.Fixtures{
			Resources: map[spec.ResourceID]interface{}{},
		},
	}
	data, err = generator.Generate(
		&spec.JSONSchema{Ref: "#/definitions/charge"}, "", nil)
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{}, data)

	// error: unhandled JSON schema type
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(
		&spec.JSONSchema{Type: []string{"string"}}, "", nil)
	assert.Equal(t,
		fmt.Errorf("Expected response to be a list or include $ref"),
		err)

	// error: no definition in OpenAPI
	generator = DataGenerator{testSpec.Definitions, testFixtures}
	data, err = generator.Generate(
		&spec.JSONSchema{Ref: "#/definitions/doesnt-exist"}, "", nil)
	assert.Equal(t,
		fmt.Errorf("Couldn't dereference: #/definitions/doesnt-exist"),
		err)
}

// ---

func TestDefinitionFromJSONPointer(t *testing.T) {
	definition, err := definitionFromJSONPointer("#/definitions/charge")
	assert.Nil(t, err)
	assert.Equal(t, "charge", definition)
}
