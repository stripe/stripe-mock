package main

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	assert "github.com/stretchr/testify/require"
)

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
				ResourceID("charge"): map[string]interface{}{"id": "ch_123"},
			},
		}

	testSpec = &OpenAPISpec{
		Definitions: map[string]OpenAPIDefinition{
			"charge": {XResourceID: "charge"},
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

// ---

func TestStubServerRouteRequest(t *testing.T) {
	server := &StubServer{spec: testSpec}
	server.initializeRouter()

	assert.Equal(t, chargeAllMethod, server.routeRequest(
		&http.Request{Method: "GET", URL: &url.URL{Path: "/v1/charges"}}))
	assert.Equal(t, chargeCreateMethod, server.routeRequest(
		&http.Request{Method: "POST", URL: &url.URL{Path: "/v1/charges"}}))
	assert.Equal(t, chargeGetMethod, server.routeRequest(
		&http.Request{Method: "GET", URL: &url.URL{Path: "/v1/charges/ch_123"}}))
	assert.Equal(t, chargeDeleteMethod, server.routeRequest(
		&http.Request{Method: "DELETE", URL: &url.URL{Path: "/v1/charges/ch_123"}}))

	assert.Equal(t, (*OpenAPIMethod)(nil), server.routeRequest(
		&http.Request{Method: "GET", URL: &url.URL{Path: "/v1/doesnt-exist"}}))
}

// ---

func TestCompilePath(t *testing.T) {
	assert.Equal(t, `/v1/charges`,
		compilePath(OpenAPIPath("/v1/charges")).String())
	assert.Equal(t, `/v1/charges/(?P<id>\w+)`,
		compilePath(OpenAPIPath("/v1/charges/{id}")).String())
}

func TestGenerateResponseData(t *testing.T) {
	var data interface{}
	var err error

	// basic reference
	data, err = generateResponseData(
		JSONSchema(map[string]interface{}{"$ref": "#/definitions/charge"}), "",
		testSpec.Definitions, testFixtures)
	assert.Nil(t, err)
	assert.Equal(t,
		testFixtures.Resources["charge"].(map[string]interface{})["id"],
		data.(map[string]interface{})["id"])

	// list
	data, err = generateResponseData(
		JSONSchema(map[string]interface{}{
			"properties": map[string]interface{}{
				"data": map[string]interface{}{
					"items": map[string]interface{}{
						"$ref": "#/definitions/charge",
					},
				},
				"object": map[string]interface{}{
					"enum": []interface{}{"list"},
				},
			},
		}), "/v1/charges",
		testSpec.Definitions, testFixtures)
	assert.Nil(t, err)
	assert.Equal(t, "list", data.(map[string]interface{})["object"])
	assert.Equal(t, "/v1/charges", data.(map[string]interface{})["url"])
	assert.Equal(t,
		testFixtures.Resources["charge"].(map[string]interface{})["id"],
		data.(map[string]interface{})["data"].([]interface{})[0].(map[string]interface{})["id"])

	// error: unhandled JSON schema type
	data, err = generateResponseData(
		JSONSchema(map[string]interface{}{}), "",
		testSpec.Definitions, testFixtures)
	assert.Equal(t,
		fmt.Errorf("Expected response to be a list or include $ref"),
		err)

	// error: no definition in OpenAPI
	data, err = generateResponseData(
		JSONSchema(map[string]interface{}{"$ref": "#/definitions/doesnt-exist"}), "",
		testSpec.Definitions, testFixtures)
	assert.Equal(t,
		fmt.Errorf("Expected definitions to include doesnt-exist"),
		err)

	// error: no fixture
	data, err = generateResponseData(
		JSONSchema(map[string]interface{}{"$ref": "#/definitions/charge"}), "",
		testSpec.Definitions,
		// this is an empty set of fixtures
		&Fixtures{
			Resources: map[ResourceID]interface{}{},
		})
	assert.Equal(t,
		fmt.Errorf("Expected fixtures to include charge"),
		err)
}

func TestDefinitionFromJSONPointer(t *testing.T) {
	definition, err := definitionFromJSONPointer("#/definitions/charge")
	assert.Nil(t, err)
	assert.Equal(t, "charge", definition)
}
