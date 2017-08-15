package main

import (
	"encoding/json"

	"github.com/stripe/stripe-mock/spec"
)

var chargeAllMethod *spec.Method
var chargeCreateMethod *spec.Method
var chargeDeleteMethod *spec.Method
var chargeGetMethod *spec.Method

// Try to avoid using the real spec as much as possible because it's more
// complicated and slower. A test spec is provided below. If you do use it,
// don't mutate it.
var realSpec spec.Spec
var realFixtures spec.Fixtures

var testSpec spec.Spec
var testFixtures spec.Fixtures

func init() {
	initRealSpec()
	initTestSpec()
}

func initRealSpec() {
	// Load the spec information from go-bindata
	data, err := Asset("openapi/openapi/spec2.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &realSpec)
	if err != nil {
		panic(err)
	}

	// And do the same for fixtures
	data, err = Asset("openapi/openapi/fixtures.json")
	if err != nil {
		panic(err)
	}

	var fixtures spec.Fixtures
	err = json.Unmarshal(data, &fixtures)
	if err != nil {
		panic(err)
	}
}

func initTestSpec() {
	chargeAllMethod = &spec.Method{}
	chargeCreateMethod = &spec.Method{
		Parameters: []*spec.Parameter{
			{
				In: "body",
				Schema: &spec.JSONSchema{
					Properties: map[string]*spec.JSONSchema{
						"amount": {
							Type: []string{"integer"},
						},
					},
					RawFields: map[string]interface{}{
						"properties": map[string]interface{}{
							"amount": map[string]interface{}{
								"type": []interface{}{
									"integer",
								},
							},
						},
						"required": []interface{}{
							"amount",
						},
					},
				},
			},
		},
		Responses: map[spec.StatusCode]spec.Response{
			"200": {
				Schema: &spec.JSONSchema{
					Ref: "#/definitions/customer",
				},
			},
		},
	}
	chargeDeleteMethod = &spec.Method{}
	chargeGetMethod = &spec.Method{}

	testFixtures =
		spec.Fixtures{
			Resources: map[spec.ResourceID]interface{}{
				spec.ResourceID("charge"): map[string]interface{}{
					"customer": "cus_123",
					"id":       "ch_123",
				},
				spec.ResourceID("customer"): map[string]interface{}{"id": "cus_123"},
			},
		}

	testSpec = spec.Spec{
		Definitions: map[string]*spec.JSONSchema{
			"charge": {
				Properties: map[string]*spec.JSONSchema{
					// Normally a customer ID, but expandable to a full
					// customer resource
					"customer": {
						Type: []string{"string"},
						XExpansionResources: &spec.JSONSchema{
							OneOf: []*spec.JSONSchema{
								{Ref: "#/definitions/customer"},
							},
						},
					},
				},
				XExpandableFields: []string{"customer"},
				XResourceID:       "charge",
			},
			"customer": {
				XResourceID: "customer",
			},
		},
		Paths: map[spec.Path]map[spec.HTTPVerb]*spec.Method{
			spec.Path("/v1/charges"): {
				"get":  chargeAllMethod,
				"post": chargeCreateMethod,
			},
			spec.Path("/v1/charges/{id}"): {
				"get":    chargeGetMethod,
				"delete": chargeDeleteMethod,
			},
		},
	}
}
