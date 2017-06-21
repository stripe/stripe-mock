package main

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var chargeAllMethod *OpenAPIMethod
var chargeCreateMethod *OpenAPIMethod
var chargeDeleteMethod *OpenAPIMethod
var chargeGetMethod *OpenAPIMethod
var testSpec *OpenAPISpec

func init() {
	chargeAllMethod = &OpenAPIMethod{}
	chargeCreateMethod = &OpenAPIMethod{}
	chargeDeleteMethod = &OpenAPIMethod{}
	chargeGetMethod = &OpenAPIMethod{}

	testSpec = &OpenAPISpec{
		Paths: map[OpenAPIPath]map[HTTPVerb]*OpenAPIMethod{
			OpenAPIPath("/v1/charges"): map[HTTPVerb]*OpenAPIMethod{
				"get":  chargeAllMethod,
				"post": chargeCreateMethod,
			},
			OpenAPIPath("/v1/charges/{id}"): map[HTTPVerb]*OpenAPIMethod{
				"get":    chargeGetMethod,
				"delete": chargeDeleteMethod,
			},
		},
	}
}

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

func TestDefinitionFromJSONPointer(t *testing.T) {
	definition, err := definitionFromJSONPointer("#/definitions/charge")
	assert.Nil(t, err)
	assert.Equal(t, "charge", definition)
}
