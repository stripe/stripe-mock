package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/stripe/stripe-mock/spec"
)

var listSchema *spec.Schema

func init() {
	listSchema = &spec.Schema{
		Type: "object",
		Properties: map[string]*spec.Schema{
			"data": {
				Items: &spec.Schema{
					Ref: "#/components/schemas/charge",
				},
			},
			"has_more": {
				Type: "boolean",
			},
			"object": {
				Enum: []interface{}{"list"},
			},
			"total_count": {
				Type: "integer",
			},
			"url": {
				Type:    "string",
				Pattern: "^/v1/charges",
			},
		},
	}
}

func TestConcurrentAccess(t *testing.T) {
	var generator DataGenerator

	// We use the real spec here because when there was a concurrency problem,
	// it wasn't revealed due to the test spec being oversimplistic.
	generator = DataGenerator{realSpec.Components.Schemas, &realFixtures}

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := generator.Generate(
				&spec.Schema{Ref: "#/components/schemas/subscription"}, "", nil)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
}

func TestGenerateResponseData(t *testing.T) {
	var data interface{}
	var err error
	var generator DataGenerator

	// basic reference
	generator = DataGenerator{testSpec.Components.Schemas, &testFixtures}
	data, err = generator.Generate(
		&spec.Schema{Ref: "#/components/schemas/charge"}, "", nil)
	assert.Nil(t, err)
	assert.Equal(t,
		testFixtures.Resources["charge"].(map[string]interface{})["id"],
		data.(map[string]interface{})["id"])

	// Makes sure that customer is *not* expanded
	assert.Equal(t,
		testFixtures.Resources["customer"].(map[string]interface{})["id"],
		data.(map[string]interface{})["customer"])

	// expansion
	generator = DataGenerator{testSpec.Components.Schemas, &testFixtures}
	data, err = generator.Generate(
		&spec.Schema{Ref: "#/components/schemas/charge"},
		"",
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{"customer": {expansions: map[string]*ExpansionLevel{}}}})
	assert.Nil(t, err)
	assert.Equal(t,
		testFixtures.Resources["customer"].(map[string]interface{})["id"],
		data.(map[string]interface{})["customer"].(map[string]interface{})["id"])

	// bad expansion
	generator = DataGenerator{testSpec.Components.Schemas, &testFixtures}
	data, err = generator.Generate(
		&spec.Schema{Ref: "#/components/schemas/charge"},
		"",
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{"id": {expansions: map[string]*ExpansionLevel{}}}})
	assert.Equal(t, err, errExpansionNotSupported)

	// bad nested expansion
	generator = DataGenerator{testSpec.Components.Schemas, &testFixtures}
	data, err = generator.Generate(
		&spec.Schema{Ref: "#/components/schemas/charge"},
		"",
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{"customer.id": {expansions: map[string]*ExpansionLevel{}}}})
	assert.Equal(t, err, errExpansionNotSupported)

	// wildcard expansion
	generator = DataGenerator{testSpec.Components.Schemas, &testFixtures}
	data, err = generator.Generate(
		&spec.Schema{Ref: "#/components/schemas/charge"},
		"",
		&ExpansionLevel{wildcard: true})
	assert.Nil(t, err)
	assert.Equal(t,
		testFixtures.Resources["customer"].(map[string]interface{})["id"],
		data.(map[string]interface{})["customer"].(map[string]interface{})["id"])

	// list
	generator = DataGenerator{testSpec.Components.Schemas, &testFixtures}
	data, err = generator.Generate(listSchema, "/v1/charges", nil)
	assert.Nil(t, err)
	assert.Equal(t, "list", data.(map[string]interface{})["object"])
	assert.Equal(t, "/v1/charges", data.(map[string]interface{})["url"])
	assert.Equal(t,
		testFixtures.Resources["charge"].(map[string]interface{})["id"],
		data.(map[string]interface{})["data"].([]interface{})[0].(map[string]interface{})["id"])

	// nested list
	generator = DataGenerator{
		testSpec.Components.Schemas,
		&spec.Fixtures{
			Resources: map[spec.ResourceID]interface{}{
				spec.ResourceID("charge"): map[string]interface{}{"id": "ch_123"},
				spec.ResourceID("with_charges_list"): map[string]interface{}{
					"charges_list": map[string]interface{}{
						"url": "/v1/charges",
					},
				},
			},
		},
	}
	data, err = generator.Generate(
		&spec.Schema{
			Type: "object",
			Properties: map[string]*spec.Schema{
				"charges_list": listSchema,
			},
			XResourceID: "with_charges_list",
		}, "", nil)
	assert.Nil(t, err)
	chargesList := data.(map[string]interface{})["charges_list"]
	assert.Equal(t, "list", chargesList.(map[string]interface{})["object"])
	assert.Equal(t, "/v1/charges", chargesList.(map[string]interface{})["url"])
	assert.Equal(t,
		testFixtures.Resources["charge"].(map[string]interface{})["id"],
		chargesList.(map[string]interface{})["data"].([]interface{})[0].(map[string]interface{})["id"])
}

func TestValidFixtures(t *testing.T) {
	// Every fixture should validate according to the schema it's a fixture for
	for name, schema := range realSpec.Components.Schemas {
		if schema.XResourceID == "" {
			continue
		}
		t.Run(name, func(t2 *testing.T) {
			fixture, ok := realFixtures.Resources[spec.ResourceID(schema.XResourceID)]
			assert.True(t2, ok)
			validator, err := spec.GetValidatorForOpenAPI3Schema(schema, realComponentsForValidation)
			assert.NoError(t2, err)
			err = validator.Validate(fixture)
			assert.NoError(t2, err)
		})
	}
}

// Tests that DataGenerator can generate an example of the given schema, and
// that the example validates against the schema correctly
func testCanGenerate(t *testing.T, path spec.Path, schema *spec.Schema, expand bool) {
	assert.NotNil(t, schema)

	generator := DataGenerator{
		definitions: realSpec.Components.Schemas,
		fixtures:    &realFixtures,
	}

	var expansions *ExpansionLevel
	if expand {
		expansions = &ExpansionLevel{
			expansions: make(map[string]*ExpansionLevel),
			wildcard:   true,
		}
	}

	var example interface{}
	var err error
	assert.NotPanics(t, func() {
		example, err = generator.Generate(schema, string(path), expansions)
	})
	assert.NoError(t, err)

	validator, err := spec.GetValidatorForOpenAPI3Schema(schema, realComponentsForValidation)
	assert.NoError(t, err)
	err = validator.Validate(example)
	if err != nil {
		t.Logf("Schema is: %s", schema)
		exampleJson, err := json.MarshalIndent(example, "", "  ")
		assert.NoError(t, err)
		t.Logf("Example is: %s", exampleJson)
	}
	assert.NoError(t, err)
}

func TestResourcesCanBeGenerated(t *testing.T) {
	for url, operations := range realSpec.Paths {
		for method, operation := range operations {
			schema := operation.Responses[spec.StatusCode("200")].Content["application/json"].Schema
			t.Run(
				fmt.Sprintf("%s %s (without expansions)", method, url),
				func(t2 *testing.T) { testCanGenerate(t2, url, schema, false) },
			)
		}
	}
}

func TestResourcesCanBeGeneratedAndExpanded(t *testing.T) {
	t.Skip("This test is known to fail because fixtures are missing for some " +
		"expandable subresources.")
	for url, operations := range realSpec.Paths {
		for method, operation := range operations {
			schema := operation.Responses[spec.StatusCode("200")].Content["application/json"].Schema
			t.Run(
				fmt.Sprintf("%s %s (with expansions)", method, url),
				func(t2 *testing.T) { testCanGenerate(t2, url, schema, true) },
			)
		}
	}
}

// ---

func TestDefinitionFromJSONPointer(t *testing.T) {
	definition := definitionFromJSONPointer("#/components/schemas/charge")
	assert.Equal(t, "charge", definition)
}
