package main

import (
	"encoding/json"

	"github.com/stripe/stripe-mock/spec"
)

var applicationFeeRefundCreateMethod *spec.Operation
var applicationFeeRefundGetMethod *spec.Operation
var chargeAllMethod *spec.Operation
var chargeCreateMethod *spec.Operation
var chargeDeleteMethod *spec.Operation
var chargeGetMethod *spec.Operation
var invoicePayMethod *spec.Operation

// Try to avoid using the real spec as much as possible because it's more
// complicated and slower. A test spec is provided below. If you do use it,
// don't mutate it.
var realSpec spec.Spec
var realFixtures spec.Fixtures
var realComponentsForValidation *spec.ComponentsForValidation

var testSpec spec.Spec
var testFixtures spec.Fixtures

func init() {
	initRealSpec()
	initTestSpec()
}

func initRealSpec() {
	// Load the spec information from go-bindata
	data, err := Asset("openapi/openapi/spec3.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &realSpec)
	if err != nil {
		panic(err)
	}

	realComponentsForValidation =
		spec.GetComponentsForValidation(&realSpec.Components)

	// And do the same for fixtures
	data, err = Asset("openapi/openapi/fixtures3.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &realFixtures)
	if err != nil {
		panic(err)
	}
}

func initTestSpec() {
	// These are basically here to give us a URL to test against that has
	// multiple parameters in it.
	applicationFeeRefundCreateMethod = &spec.Operation{}
	applicationFeeRefundGetMethod = &spec.Operation{}

	chargeAllMethod = &spec.Operation{}
	chargeCreateMethod = &spec.Operation{
		RequestBody: &spec.RequestBody{
			Content: map[string]spec.MediaType{
				"application/x-www-form-urlencoded": {
					Schema: &spec.Schema{
						Properties: map[string]*spec.Schema{
							"amount": {
								Type: "integer",
							},
						},
						Required: []string{"amount"},
					},
				},
			},
		},
		Responses: map[spec.StatusCode]spec.Response{
			"200": {
				Content: map[string]spec.MediaType{
					"application/json": {
						Schema: &spec.Schema{
							Ref: "#/components/schemas/customer",
						},
					},
				},
			},
		},
	}
	chargeDeleteMethod = &spec.Operation{}
	chargeGetMethod = &spec.Operation{}

	// Here so we can test the relatively rare "action" operations (e.g.,
	// `POST` to `/pay` on an invoice).
	invoicePayMethod = &spec.Operation{}

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
		Components: spec.Components{
			Schemas: map[string]*spec.Schema{
				"charge": {
					Type: "object",
					Properties: map[string]*spec.Schema{
						"id": {Type: "string"},
						// Normally a customer ID, but expandable to a full
						// customer resource
						"customer": {
							AnyOf: []*spec.Schema{
								{Type: "string"},
								{Ref: "#/components/schemas/customer"},
							},
							XExpansionResources: &spec.ExpansionResources{
								OneOf: []*spec.Schema{
									{Ref: "#/components/schemas/customer"},
								},
							},
						},
					},
					XExpandableFields: &[]string{"customer"},
					XResourceID:       "charge",
				},
				"customer": {
					Type:        "object",
					XResourceID: "customer",
				},
			},
		},
		Paths: map[spec.Path]map[spec.HTTPVerb]*spec.Operation{
			spec.Path("/v1/application_fees/{fee}/refunds"): {
				"get": applicationFeeRefundCreateMethod,
			},
			spec.Path("/v1/application_fees/{fee}/refunds/{id}"): {
				"get": applicationFeeRefundGetMethod,
			},
			spec.Path("/v1/charges"): {
				"get":  chargeAllMethod,
				"post": chargeCreateMethod,
			},
			spec.Path("/v1/charges/{id}"): {
				"get":    chargeGetMethod,
				"delete": chargeDeleteMethod,
			},
			spec.Path("/v1/invoices/{id}/pay"): {
				"post": invoicePayMethod,
			},
		},
	}
}
