package main

import (
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
			"has_more": nil,
			"object": {
				Enum: []string{"list"},
			},
			"total_count": nil,
			"url":         nil,
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
						"url": "/v1/from_charges_list",
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
	assert.Equal(t, "/v1/from_charges_list", chargesList.(map[string]interface{})["url"])
	assert.Equal(t,
		testFixtures.Resources["charge"].(map[string]interface{})["id"],
		chargesList.(map[string]interface{})["data"].([]interface{})[0].(map[string]interface{})["id"])
}

// ---

func TestDefinitionFromJSONPointer(t *testing.T) {
	definition := definitionFromJSONPointer("#/components/schemas/charge")
	assert.Equal(t, "charge", definition)
}
