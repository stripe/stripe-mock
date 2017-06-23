package main

import ()

var chargeAllMethod *OpenAPIMethod
var chargeCreateMethod *OpenAPIMethod
var chargeDeleteMethod *OpenAPIMethod
var chargeGetMethod *OpenAPIMethod
var testSpec *OpenAPISpec
var testFixtures *Fixtures

func init() {
	chargeAllMethod = &OpenAPIMethod{}
	chargeCreateMethod = &OpenAPIMethod{}
	chargeDeleteMethod = &OpenAPIMethod{}
	chargeGetMethod = &OpenAPIMethod{}

	testFixtures =
		&Fixtures{
			Resources: map[ResourceID]interface{}{
				ResourceID("charge"):   map[string]interface{}{"id": "ch_123"},
				ResourceID("customer"): map[string]interface{}{"id": "cus_123"},
			},
		}

	testSpec = &OpenAPISpec{
		Definitions: map[string]*JSONSchema{
			"charge": {
				Properties: map[string]*JSONSchema{
					// Normally a customer ID, but expandable to a full
					// customer resource
					"customer": {
						Type: []string{"string"},
						XExpansionResources: &JSONSchema{
							OneOf: []*JSONSchema{
								&JSONSchema{Ref: "#/definitions/customer"},
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
		Paths: map[OpenAPIPath]map[HTTPVerb]*OpenAPIMethod{
			OpenAPIPath("/v1/charges"): {
				"get":  chargeAllMethod,
				"post": chargeCreateMethod,
			},
			OpenAPIPath("/v1/charges/{id}"): {
				"get":    chargeGetMethod,
				"delete": chargeDeleteMethod,
			},
		},
	}
}
