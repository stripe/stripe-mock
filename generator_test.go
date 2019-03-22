package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
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

//
// Tests
//

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
			_, err := generator.Generate(&GenerateParams{
				Schema: &spec.Schema{Ref: "#/components/schemas/subscription"},
			})
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
}

func TestGenerateResponseData(t *testing.T) {
	// basic reference
	{
		generator := DataGenerator{testSpec.Components.Schemas, &testFixtures}
		data, err := generator.Generate(&GenerateParams{
			Schema: &spec.Schema{Ref: "#/components/schemas/charge"},
		})
		assert.Nil(t, err)
		assert.Equal(t,
			testFixtures.Resources["charge"].(map[string]interface{})["object"],
			data.(map[string]interface{})["object"])

		// Makes sure that customer is *not* expanded
		assert.Equal(t,
			testFixtures.Resources["customer"].(map[string]interface{})["id"],
			data.(map[string]interface{})["customer"])
	}

	// expansion
	{
		generator := DataGenerator{testSpec.Components.Schemas, &testFixtures}
		data, err := generator.Generate(&GenerateParams{
			Expansions: &ExpansionLevel{
				expansions: map[string]*ExpansionLevel{"customer": {
					expansions: map[string]*ExpansionLevel{}},
				},
			},
			Schema: &spec.Schema{Ref: "#/components/schemas/charge"},
		})

		assert.Nil(t, err)
		assert.Equal(t,
			testFixtures.Resources["customer"].(map[string]interface{})["id"],
			data.(map[string]interface{})["customer"].(map[string]interface{})["id"])
	}

	// bad expansion
	{
		generator := DataGenerator{testSpec.Components.Schemas, &testFixtures}
		_, err := generator.Generate(&GenerateParams{
			Expansions: &ExpansionLevel{
				expansions: map[string]*ExpansionLevel{"id": {
					expansions: map[string]*ExpansionLevel{}},
				},
			},
			Schema: &spec.Schema{Ref: "#/components/schemas/charge"},
		})

		assert.Equal(t, err, errExpansionNotSupported)
	}

	// bad nested expansion
	{
		generator := DataGenerator{testSpec.Components.Schemas, &testFixtures}
		_, err := generator.Generate(&GenerateParams{
			Expansions: &ExpansionLevel{
				expansions: map[string]*ExpansionLevel{"customer.id": {
					expansions: map[string]*ExpansionLevel{}},
				},
			},
			Schema: &spec.Schema{Ref: "#/components/schemas/charge"},
		})
		assert.Equal(t, err, errExpansionNotSupported)
	}

	// wildcard expansion
	{
		generator := DataGenerator{testSpec.Components.Schemas, &testFixtures}
		data, err := generator.Generate(&GenerateParams{
			Expansions: &ExpansionLevel{wildcard: true},
			Schema:     &spec.Schema{Ref: "#/components/schemas/charge"},
		})
		assert.Nil(t, err)
		assert.Equal(t,
			testFixtures.Resources["customer"].(map[string]interface{})["id"],
			data.(map[string]interface{})["customer"].(map[string]interface{})["id"])
	}

	// list
	{
		generator := DataGenerator{testSpec.Components.Schemas, &testFixtures}
		data, err := generator.Generate(&GenerateParams{
			RequestPath: "/v1/charges",
			Schema:      listSchema,
		})
		assert.Nil(t, err)
		assert.Equal(t, "list", data.(map[string]interface{})["object"])
		assert.Equal(t, "/v1/charges", data.(map[string]interface{})["url"])
		assert.Equal(t,
			testFixtures.Resources["charge"].(map[string]interface{})["id"],
			data.(map[string]interface{})["data"].([]interface{})[0].(map[string]interface{})["id"])
	}

	// nested list
	{
		generator := DataGenerator{
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
		data, err := generator.Generate(&GenerateParams{
			Schema: &spec.Schema{
				Type: "object",
				Properties: map[string]*spec.Schema{
					"charges_list": listSchema,
				},
				XResourceID: "with_charges_list",
			},
		})
		assert.Nil(t, err)
		chargesList := data.(map[string]interface{})["charges_list"]
		assert.Equal(t, "list", chargesList.(map[string]interface{})["object"])
		assert.Equal(t, "/v1/charges", chargesList.(map[string]interface{})["url"])
		assert.Equal(t,
			testFixtures.Resources["charge"].(map[string]interface{})["id"],
			chargesList.(map[string]interface{})["data"].([]interface{})[0].(map[string]interface{})["id"])
	}

	// generated primary ID
	{
		generator := DataGenerator{testSpec.Components.Schemas, &spec.Fixtures{
			Resources: map[spec.ResourceID]interface{}{
				spec.ResourceID("charge"): map[string]interface{}{
					"id": "ch_123",
				},
			},
		}}
		data, err := generator.Generate(&GenerateParams{
			PathParams: &PathParamsMap{},
			Schema:     &spec.Schema{Ref: "#/components/schemas/charge"},
		})
		assert.Nil(t, err)

		// Should not be equal to our fixture's ID because it's generated.
		assert.NotEqual(t, "ch_123",
			data.(map[string]interface{})["id"])

		// However, the fixture's ID's prefix will have been used to generate
		// the new ID, so it should share a prefix.
		assert.Regexp(t, regexp.MustCompile("^ch_"),
			data.(map[string]interface{})["id"])
	}

	// generated primary ID (nil PathParamsMap)
	{
		generator := DataGenerator{testSpec.Components.Schemas, &spec.Fixtures{
			Resources: map[spec.ResourceID]interface{}{
				spec.ResourceID("charge"): map[string]interface{}{
					"id": "ch_123",
				},
			},
		}}
		data, err := generator.Generate(&GenerateParams{
			PathParams: nil,
			Schema:     &spec.Schema{Ref: "#/components/schemas/charge"},
		})
		assert.Nil(t, err)

		// Should not be equal to our fixture's ID because it's generated.
		assert.NotEqual(t, "ch_123",
			data.(map[string]interface{})["id"])

		// However, the fixture's ID's prefix will have been used to generate
		// the new ID, so it should share a prefix.
		assert.Regexp(t, regexp.MustCompile("^ch_"),
			data.(map[string]interface{})["id"])
	}

	// injected ID
	{
		generator := DataGenerator{testSpec.Components.Schemas, &spec.Fixtures{
			Resources: map[spec.ResourceID]interface{}{
				spec.ResourceID("charge"): map[string]interface{}{
					// This is contrived, but we inject the value we expect to be
					// replaced into `customer` as well so that we can verify the
					// secondary behavior that replaces all values that look like a
					// replaced ID (as well as the ID).
					"customer": "ch_123",

					"id": "ch_123",
				},
			},
		}}
		newID := "ch_123_InjectedFromURL"
		data, err := generator.Generate(&GenerateParams{
			PathParams: &PathParamsMap{PrimaryID: &newID},
			Schema:     &spec.Schema{Ref: "#/components/schemas/charge"},
		})
		assert.Nil(t, err)
		assert.Equal(t,
			newID,
			data.(map[string]interface{})["id"])
		assert.Equal(t,
			newID,
			data.(map[string]interface{})["customer"])
	}

	// injected secondary ID
	{
		generator := DataGenerator{testSpec.Components.Schemas, &spec.Fixtures{
			Resources: map[spec.ResourceID]interface{}{
				spec.ResourceID("charge"): map[string]interface{}{
					"id": "ch_123",
				},
				spec.ResourceID("customer"): map[string]interface{}{
					"id":     "cus_123",
					"object": "customer",
				},
			},
		}}
		newCustomerID := "cus_123_InjectedFromURL"
		data, err := generator.Generate(&GenerateParams{
			Expansions: &ExpansionLevel{
				expansions: map[string]*ExpansionLevel{"customer": {
					expansions: map[string]*ExpansionLevel{}},
				},
			},
			PathParams: &PathParamsMap{
				SecondaryIDs: []*PathParamsSecondaryID{
					{ID: newCustomerID, Name: "customer"},
				},
			},
			Schema: &spec.Schema{Ref: "#/components/schemas/charge"},
		})
		assert.Nil(t, err)

		// The top level ID. It will have been randomized, but should still be
		// a charge.
		assert.Regexp(t, regexp.MustCompile("^ch_"),
			data.(map[string]interface{})["id"])

		// The nested customer ID. This should have changed.
		assert.Equal(t,
			map[string]interface{}{
				"id":     newCustomerID,
				"object": "customer",
			},
			data.(map[string]interface{})["customer"])
	}

	// data replacement on `POST`
	{
		generator := DataGenerator{testSpec.Components.Schemas, &testFixtures}
		data, err := generator.Generate(&GenerateParams{
			RequestData: map[string]interface{}{
				"customer": "cus_9999",
			},
			RequestMethod: http.MethodPost,
			Schema:        &spec.Schema{Ref: "#/components/schemas/charge"},
		})
		assert.Nil(t, err)
		assert.Equal(t,
			"cus_9999",
			data.(map[string]interface{})["customer"])
	}

	// *no* data replacement on non-`POST`
	{
		generator := DataGenerator{testSpec.Components.Schemas, &testFixtures}
		data, err := generator.Generate(&GenerateParams{
			RequestData: map[string]interface{}{
				"customer": "cus_9999",
			},
			RequestMethod: http.MethodGet,
			Schema:        &spec.Schema{Ref: "#/components/schemas/charge"},
		})
		assert.Nil(t, err)
		assert.NotEqual(t,
			"cus_9999",
			data.(map[string]interface{})["customer"])
	}

	// synthetic schema
	{
		generator := DataGenerator{testSpec.Components.Schemas, &testFixtures}
		data, err := generator.Generate(&GenerateParams{
			Schema: &spec.Schema{
				Properties: map[string]*spec.Schema{
					"string_property": {
						Type: spec.TypeString,
					},
				},
				Required:    []string{"string_property"},
				Type:        spec.TypeObject,
				XResourceID: "",
			},
		})
		assert.Nil(t, err)
		assert.Equal(t,
			map[string]interface{}{
				"string_property": "",
			},
			data,
		)
	}

	// pick non-deleted anyOf branch
	{
		generator := DataGenerator{testSpec.Components.Schemas, &testFixtures}
		data, err := generator.Generate(&GenerateParams{
			// Just needs to be any HTTP method that's not DELETE
			RequestMethod: http.MethodPost,
			Schema: &spec.Schema{AnyOf: []*spec.Schema{
				// put the deleted version first so we know it's not just
				// returning the first result
				{Ref: "#/components/schemas/deleted_customer"},
				{Ref: "#/components/schemas/customer"},
			}},
		})
		assert.Nil(t, err)

		// There should be no deleted field
		_, ok := data.(map[string]interface{})["deleted"]
		assert.False(t, ok)
	}

	// pick deleted anyOf branch
	{
		generator := DataGenerator{testSpec.Components.Schemas, &testFixtures}
		data, err := generator.Generate(&GenerateParams{
			RequestMethod: http.MethodDelete,
			Schema: &spec.Schema{AnyOf: []*spec.Schema{
				// put the non-deleted version first so we know it's not just
				// returning the first result
				{Ref: "#/components/schemas/customer"},
				{Ref: "#/components/schemas/deleted_customer"},
			}},
		})
		assert.Nil(t, err)

		// There should be a deleted field
		_, ok := data.(map[string]interface{})["deleted"]
		assert.True(t, ok)
	}
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

//
// Tests for private functions
//

func TestDefinitionFromJSONPointer(t *testing.T) {
	definition := definitionFromJSONPointer("#/components/schemas/charge")
	assert.Equal(t, "charge", definition)
}

// This is meant as quite a blunt test. See TestMaybeReplaceID for something
// that's probably easier to introspect/debug.
func TestDistributeReplacedIDs(t *testing.T) {
	newID := "new-id"
	oldID := "old-id"

	newSecondaryID := "new-secondary-id"
	oldSecondaryID := "old-secondary-id"

	pathParams := &PathParamsMap{
		PrimaryID:         &newID,
		replacedPrimaryID: &oldID,

		SecondaryIDs: []*PathParamsSecondaryID{
			{
				ID:          newSecondaryID,
				replacedIDs: []string{oldSecondaryID},
			},
		},
	}

	data := map[string]interface{}{
		"key_with_id":   oldID,
		"unrelated_key": "foo",
		"nested": map[string]interface{}{
			"key_with_secondary_id": oldSecondaryID,
			"slice": []interface{}{
				map[string]interface{}{
					"key_with_id":   oldID,
					"unrelated_key": "foo",
					"url":           "/v1/charges/" + oldID + "/refunds",
				},
			},
		},
	}

	// This function modifies structure in place.
	distributeReplacedIDs(pathParams, data)

	assert.Equal(t,
		map[string]interface{}{
			"key_with_id":   newID,
			"unrelated_key": "foo",
			"nested": map[string]interface{}{
				"key_with_secondary_id": newSecondaryID,
				"slice": []interface{}{
					map[string]interface{}{
						"key_with_id":   newID,
						"unrelated_key": "foo",
						"url":           "/v1/charges/" + newID + "/refunds",
					},
				},
			},
		},
		data,
	)
}

func TestDistributeReplacedIDsInURL(t *testing.T) {
	newID := "new-id"
	oldID := "old-id"

	newSecondaryID := "new-secondary-id"
	oldSecondaryID := "old-secondary-id"

	pathParams := &PathParamsMap{
		PrimaryID:         &newID,
		replacedPrimaryID: &oldID,

		SecondaryIDs: []*PathParamsSecondaryID{
			{
				ID:          newSecondaryID,
				replacedIDs: []string{oldSecondaryID},
			},
		},
	}

	// Replaces primary ID
	{
		data, ok := distributeReplacedIDsInURL(pathParams, "/v1/charges/"+oldID+"/refunds")
		assert.True(t, ok)
		assert.Equal(t, "/v1/charges/"+newID+"/refunds", data)
	}

	// Replaces secondary ID
	{
		data, ok := distributeReplacedIDsInURL(pathParams, "/v1/charges/"+oldSecondaryID+"/refunds")
		assert.True(t, ok)
		assert.Equal(t, "/v1/charges/"+newSecondaryID+"/refunds", data)
	}

	// Doesn't replace a value that's not present
	{
		_, ok := distributeReplacedIDsInURL(pathParams, "/v1/charges/other/refunds")
		assert.False(t, ok)
	}

	// Works fine on a non-string data type (by ignoring it)
	{
		_, ok := distributeReplacedIDsInURL(pathParams, 123)
		assert.False(t, ok)
	}
}

func TestDistributeReplacedIDsInValue(t *testing.T) {
	newID := "new-id"
	oldID := "old-id"

	newSecondaryID := "new-secondary-id"
	oldSecondaryID := "old-secondary-id"

	pathParams := &PathParamsMap{
		PrimaryID:         &newID,
		replacedPrimaryID: &oldID,

		SecondaryIDs: []*PathParamsSecondaryID{
			{
				ID:          newSecondaryID,
				replacedIDs: []string{oldSecondaryID},
			},
		},
	}

	// Replaces primary ID
	{
		data, ok := distributeReplacedIDsInValue(pathParams, oldID)
		assert.True(t, ok)
		assert.Equal(t, newID, data)
	}

	// Replaces secondary ID
	{
		data, ok := distributeReplacedIDsInValue(pathParams, oldSecondaryID)
		assert.True(t, ok)
		assert.Equal(t, newSecondaryID, data)
	}

	// Doesn't replace a value that's not present
	{
		_, ok := distributeReplacedIDsInValue(pathParams, "other")
		assert.False(t, ok)
	}

	// Works fine on a non-string data type (by ignoring it)
	{
		_, ok := distributeReplacedIDsInValue(pathParams, 123)
		assert.False(t, ok)
	}
}

func TestFindAnyOfBranch(t *testing.T) {
	deletedSchema := &spec.Schema{
		Properties: map[string]*spec.Schema{
			"deleted": nil,
		},
	}

	nonDeletedSchema := &spec.Schema{}

	schema := &spec.Schema{
		AnyOf: []*spec.Schema{
			deletedSchema,
			nonDeletedSchema,
		},
	}

	generator := DataGenerator{nil, nil}

	// Finds a deleted schema branch
	{
		anyOfSchema, err := generator.findAnyOfBranch(schema, true)
		assert.NoError(t, err)
		assert.Equal(t, deletedSchema, anyOfSchema)
	}

	// Finds a non-deleted schema branch
	{
		anyOfSchema, err := generator.findAnyOfBranch(schema, false)
		assert.NoError(t, err)
		assert.Equal(t, nonDeletedSchema, anyOfSchema)
	}

	// Safe to use on an empty schema
	{
		anyOfSchema, err := generator.findAnyOfBranch(&spec.Schema{}, false)
		assert.NoError(t, err)
		assert.Equal(t, (*spec.Schema)(nil), anyOfSchema)
	}
}

func TestGenerateSyntheticFixture(t *testing.T) {
	// Scalars (and an array, which is easy)
	assert.Equal(t, []string{}, generateSyntheticFixture(&spec.Schema{Type: spec.TypeArray}, ""))
	assert.Equal(t, true, generateSyntheticFixture(&spec.Schema{Type: spec.TypeBoolean}, ""))
	assert.Equal(t, 0, generateSyntheticFixture(&spec.Schema{Type: spec.TypeInteger}, ""))
	assert.Equal(t, 0.0, generateSyntheticFixture(&spec.Schema{Type: spec.TypeNumber}, ""))
	assert.Equal(t, "", generateSyntheticFixture(&spec.Schema{Type: spec.TypeString}, ""))

	// Nullable property
	assert.Equal(t, nil, generateSyntheticFixture(&spec.Schema{
		Nullable: true,
		Type:     spec.TypeString,
	}, ""))

	// Property with enum
	assert.Equal(t, "list", generateSyntheticFixture(&spec.Schema{
		Enum: []interface{}{"list"},
		Type: spec.TypeString,
	}, ""))

	// Takes the first non-reference branch of an anyOf
	assert.Equal(t, "", generateSyntheticFixture(&spec.Schema{
		AnyOf: []*spec.Schema{
			{Ref: "#/components/schemas/radar_rule"},
			{Type: spec.TypeString},
		},
	}, ""))

	// Object
	assert.Equal(t,
		map[string]interface{}{
			"has_more": true,
			"object":   "list",
			"url":      "",
		},
		generateSyntheticFixture(&spec.Schema{
			Type: "object",
			Properties: map[string]*spec.Schema{
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
					Type: "string",
				},
			},
			Required: []string{
				"has_more",
				"object",
				"url",
			},
		}, ""),
	)
}

func TestPropertyNames(t *testing.T) {
	assert.Equal(t, "bar, foo", propertyNames(&spec.Schema{
		Properties: map[string]*spec.Schema{
			"foo": nil,
			"bar": nil,
		},
	}))
	assert.Equal(t, "", propertyNames(&spec.Schema{}))
}

func TestIsDeletedResource(t *testing.T) {
	assert.True(t, isDeletedResource(&spec.Schema{
		Properties: map[string]*spec.Schema{
			"deleted": nil,
		},
	}))

	assert.False(t, isDeletedResource(&spec.Schema{}))
}

// This is meant as quite a blunt test. See TestMaybeReplaceID for something
// that's probably easier to introspect/debug.
func TestReplaceIDs(t *testing.T) {
	newID := "new-id"
	oldID := "old-id"

	newApplicationID := "new-application-id"
	oldApplicationID := "old-application-id"

	newChargeID := "new-charge-id"
	oldChargeID := "old-charge-id"

	// The generator handles multiple values for the same type of entity. Here
	// we're replacing two separate old charge IDs with a single new charge ID.
	otherOldChargeID := "other-charge-id"

	newRefundID := "new-refund-id"
	oldRefundID := "old-refund-id"

	newSourceID := "new-source-id"
	oldSourceID := "old-source-id"

	pathParams := &PathParamsMap{
		PrimaryID:         &newID,
		replacedPrimaryID: &oldID,

		SecondaryIDs: []*PathParamsSecondaryID{
			{
				ID:          newApplicationID,
				Name:        "application",
				replacedIDs: nil,
			},
			{
				ID:          newChargeID,
				Name:        "charge",
				replacedIDs: nil,
			},
			{
				ID:          newRefundID,
				Name:        "refund",
				replacedIDs: nil,
			},
			{
				ID:          newSourceID,
				Name:        "source",
				replacedIDs: nil,
			},
		},
	}

	data := map[string]interface{}{
		"id": oldID,
		"embedded_charge": map[string]interface{}{
			"id":          oldChargeID,
			"object":      "charge",
			"application": oldApplicationID,
			"embedded_refunds": []interface{}{
				map[string]interface{}{
					"id":     oldRefundID,
					"object": "refund",
				},
			},
			"source": map[string]interface{}{
				"id":     oldSourceID,
				"object": "not-source",
			},
		},
		"charge": otherOldChargeID,
		"not_replaced": map[string]interface{}{
			"id":     "other-id",
			"object": "other",
		},
	}

	// This function modifies structure in place.
	pathParams = recordAndReplaceIDs(pathParams, data)

	assert.Equal(t, oldID, *pathParams.replacedPrimaryID)

	assert.Equal(t, 1, len(pathParams.SecondaryIDs[0].replacedIDs))
	assert.Contains(t, pathParams.SecondaryIDs[0].replacedIDs, oldApplicationID)

	assert.Equal(t, 2, len(pathParams.SecondaryIDs[1].replacedIDs))
	assert.Contains(t, pathParams.SecondaryIDs[1].replacedIDs, oldChargeID)
	assert.Contains(t, pathParams.SecondaryIDs[1].replacedIDs, otherOldChargeID)

	assert.Equal(t, 1, len(pathParams.SecondaryIDs[2].replacedIDs))
	assert.Contains(t, pathParams.SecondaryIDs[2].replacedIDs, oldRefundID)

	assert.Equal(t, 1, len(pathParams.SecondaryIDs[3].replacedIDs))
	assert.Contains(t, pathParams.SecondaryIDs[3].replacedIDs, oldSourceID)

	assert.Equal(t,
		map[string]interface{}{
			"id": newID,
			"embedded_charge": map[string]interface{}{
				"id":          newChargeID,
				"object":      "charge",
				"application": newApplicationID,
				"embedded_refunds": []interface{}{
					map[string]interface{}{
						"id":     newRefundID,
						"object": "refund",
					},
				},
				"source": map[string]interface{}{
					"id":     newSourceID,
					"object": "not-source",
				},
			},
			"charge": newChargeID,
			"not_replaced": map[string]interface{}{
				"id":     "other-id",
				"object": "other",
			},
		},
		data,
	)
}

func TestStringOrEmpty(t *testing.T) {
	assert.Equal(t, "foo", stringOrEmpty("foo"))
	assert.Equal(t, "(empty)", stringOrEmpty(""))
}

//
// Private functions
//

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
		example, err = generator.Generate(&GenerateParams{
			Expansions:  expansions,
			RequestPath: string(path),
			Schema:      schema,
		})
	})
	assert.NoError(t, err)

	validator, err := spec.GetValidatorForOpenAPI3Schema(schema, realComponentsForValidation)
	assert.NoError(t, err)
	err = validator.Validate(example)
	if err != nil {
		t.Logf("Schema is: %s", schema)
		exampleJSON, err := json.MarshalIndent(example, "", "  ")
		assert.NoError(t, err)
		t.Logf("Example is: %s", exampleJSON)
	}
	assert.NoError(t, err)
}
