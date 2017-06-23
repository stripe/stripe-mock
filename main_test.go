package main

import (
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
		Definitions: map[string]JSONSchema{
			"charge": {"x-resourceId": "charge"},
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
	assert.Equal(t, `\A/v1/charges\z`,
		compilePath(OpenAPIPath("/v1/charges")).String())
	assert.Equal(t, `\A/v1/charges/(?P<id>\w+)\z`,
		compilePath(OpenAPIPath("/v1/charges/{id}")).String())
}
