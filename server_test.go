package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/brandur/stripelocal/spec"
	assert "github.com/stretchr/testify/require"
)

func TestStubServer_SetsSpecialHeaders(t *testing.T) {
	server := getStubServer(t)

	// Does this regardless of endpoint
	req := httptest.NewRequest("GET", "https://stripe.com/", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, version, resp.Header.Get("Stripelocal-Version"))
}

func TestStubServer_ParameterValidation(t *testing.T) {
	server := getStubServer(t)

	req := httptest.NewRequest("POST", "https://stripe.com/v1/charges", nil)
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Contains(t, string(body), "property 'amount' is required")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestStubServer_RoutesRequest(t *testing.T) {
	server := getStubServer(t)
	var route *stubServerRoute

	route = server.routeRequest(
		&http.Request{Method: "GET", URL: &url.URL{Path: "/v1/charges"}})
	assert.NotNil(t, route)
	assert.Equal(t, chargeAllMethod, route.method)

	route = server.routeRequest(
		&http.Request{Method: "POST", URL: &url.URL{Path: "/v1/charges"}})
	assert.NotNil(t, route)
	assert.Equal(t, chargeCreateMethod, route.method)

	route = server.routeRequest(
		&http.Request{Method: "GET", URL: &url.URL{Path: "/v1/charges/ch_123"}})
	assert.NotNil(t, route)
	assert.Equal(t, chargeGetMethod, route.method)

	route = server.routeRequest(
		&http.Request{Method: "DELETE", URL: &url.URL{Path: "/v1/charges/ch_123"}})
	assert.NotNil(t, route)
	assert.Equal(t, chargeDeleteMethod, route.method)

	route = server.routeRequest(
		&http.Request{Method: "GET", URL: &url.URL{Path: "/v1/doesnt-exist"}})
	assert.Equal(t, (*stubServerRoute)(nil), route)
}

// ---

func TestCompilePath(t *testing.T) {
	assert.Equal(t, `\A/v1/charges\z`,
		compilePath(spec.Path("/v1/charges")).String())
	assert.Equal(t, `\A/v1/charges/(?P<id>[\w-_.]+)\z`,
		compilePath(spec.Path("/v1/charges/{id}")).String())
}

func TestGetValidator(t *testing.T) {
	method := &spec.Method{Parameters: []*spec.Parameter{
		{Schema: &spec.JSONSchema{
			RawFields: map[string]interface{}{
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
			},
		}},
	}}
	validator, err := getValidator(method)
	assert.NoError(t, err)
	assert.NotNil(t, validator)

	goodData := map[string]interface{}{
		"name": "foo",
	}
	assert.NoError(t, validator.Validate(goodData))

	badData := map[string]interface{}{
		"name": 7,
	}
	assert.Error(t, validator.Validate(badData))
}

func TestGetValidator_NoSuitableParameter(t *testing.T) {
	method := &spec.Method{Parameters: []*spec.Parameter{
		{Schema: nil},
	}}
	validator, err := getValidator(method)
	assert.NoError(t, err)
	assert.Nil(t, validator)
}

func TestParseExpansionLevel(t *testing.T) {
	assert.Equal(t,
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{
			"charge":   nil,
			"customer": nil,
		}},
		ParseExpansionLevel([]string{"charge", "customer"}))

	assert.Equal(t,
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{
			"charge": {expansions: map[string]*ExpansionLevel{
				"customer": nil,
				"source":   nil,
			}},
			"customer": nil,
		}},
		ParseExpansionLevel([]string{"charge.customer", "customer", "charge.source"}))

	assert.Equal(t,
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{
			"charge": {expansions: map[string]*ExpansionLevel{
				"customer": {expansions: map[string]*ExpansionLevel{
					"default_source": nil,
				}},
			}},
		}},
		ParseExpansionLevel([]string{"charge.customer.default_source", "charge"}))

	assert.Equal(t,
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{}, wildcard: true},
		ParseExpansionLevel([]string{"*"}))
}

//
// ---
//

func getStubServer(t *testing.T) *StubServer {
	server := &StubServer{spec: testSpec}
	err := server.initializeRouter()
	assert.NoError(t, err)
	return server
}
