package spec

//TestAPIVersion is the version of the API used by the embedded testing spec.
const TestAPIVersion = "2019-01-01"

//Test returns a test pair of Spec & Fixtures.
func Test() (Spec, Fixtures) {
	// These are basically here to give us a URL to test against that has
	// multiple parameters in it.
	applicationFeeRefundCreateMethod := &Operation{}
	applicationFeeRefundGetMethod := &Operation{}

	chargeAllMethod := &Operation{
		Parameters: []*Parameter{
			{
				In:       ParameterQuery,
				Name:     "limit",
				Required: false,
				Schema: &Schema{
					Type: TypeInteger,
				},
			},
		},
		Responses: map[StatusCode]Response{
			"200": {
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type: TypeObject,
						},
					},
				},
			},
		},
	}
	chargeCreateMethod := &Operation{
		RequestBody: &RequestBody{
			Content: map[string]MediaType{
				"application/x-www-form-urlencoded": {
					Schema: &Schema{
						AdditionalProperties: false,
						Properties: map[string]*Schema{
							"amount": {
								Type: TypeInteger,
							},
						},
						Required: []string{"amount"},
					},
				},
			},
		},
		Responses: map[StatusCode]Response{
			"200": {
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/charge",
						},
					},
				},
			},
		},
	}
	chargeGetMethod := &Operation{}

	customerDeleteMethod := &Operation{
		RequestBody: &RequestBody{
			Content: map[string]MediaType{
				"application/x-www-form-urlencoded": {
					Schema: &Schema{
						AdditionalProperties: false,
						Type:                 TypeObject,
					},
				},
			},
		},
		Responses: map[StatusCode]Response{
			"200": {
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/deleted_customer",
						},
					},
				},
			},
		},
	}

	// Here so we can test the relatively rare "action" operations (e.g.,
	// `POST` to `/pay` on an invoice).
	invoicePayMethod := &Operation{}

	return Spec{
			Info: &Info{
				Version: TestAPIVersion,
			},
			Components: Components{
				Schemas: map[string]*Schema{
					"charge": {
						Type: "object",
						Properties: map[string]*Schema{
							"id": {Type: "string"},
							// Normally a customer ID, but expandable to a full
							// customer resource
							"customer": {
								AnyOf: []*Schema{
									{Type: "string"},
									{Ref: "#/components/schemas/customer"},
								},
								XExpansionResources: &ExpansionResources{
									OneOf: []*Schema{
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
					"deleted_customer": {
						Properties: map[string]*Schema{
							"deleted": {Type: "boolean"},
						},
						Type:        "object",
						XResourceID: "deleted_customer",
					},
				},
			},
			Paths: map[Path]map[HTTPVerb]*Operation{
				Path("/v1/application_fees/{fee}/refunds"): {
					"get": applicationFeeRefundCreateMethod,
				},
				Path("/v1/application_fees/{fee}/refunds/{id}"): {
					"get": applicationFeeRefundGetMethod,
				},
				Path("/v1/charges"): {
					"get":  chargeAllMethod,
					"post": chargeCreateMethod,
				},
				Path("/v1/charges/{id}"): {
					"get": chargeGetMethod,
				},
				Path("/v1/customers/{id}"): {
					"delete": customerDeleteMethod,
				},
				Path("/v1/invoices/{id}/pay"): {
					"post": invoicePayMethod,
				},
			},
		}, Fixtures{
			Resources: map[ResourceID]interface{}{
				ResourceID("charge"): map[string]interface{}{
					"customer": "cus_123",
					"id":       "ch_123",
				},
				ResourceID("customer"): map[string]interface{}{
					"id": "cus_123",
				},
				ResourceID("deleted_customer"): map[string]interface{}{
					"deleted": true,
				},
			},
		}
}
