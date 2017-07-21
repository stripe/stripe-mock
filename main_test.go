package main

import (
	"github.com/brandur/stripestub/spec"
)

var chargeAllMethod *spec.Method
var chargeCreateMethod *spec.Method
var chargeDeleteMethod *spec.Method
var chargeGetMethod *spec.Method
var testSpec *spec.Spec
var testFixtures *spec.Fixtures

func init() {
	chargeAllMethod = &spec.Method{}
	chargeCreateMethod = &spec.Method{
		Parameters: []*spec.Parameter{
			{
				In: "body",
				Schema: &spec.JSONSchema{
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
		&spec.Fixtures{
			Resources: map[spec.ResourceID]interface{}{
				spec.ResourceID("charge"): map[string]interface{}{
					"customer": "cus_123",
					"id":       "ch_123",
				},
				spec.ResourceID("customer"): map[string]interface{}{"id": "cus_123"},
			},
		}

	testSpec = &spec.Spec{
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
