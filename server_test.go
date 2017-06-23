package main

import (
	"net/http"
	"net/url"
	"testing"

	assert "github.com/stretchr/testify/require"
)

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

func TestParseExpansionLevel(t *testing.T) {
	assert.Equal(t,
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{
			"charge":   nil,
			"customer": nil,
		}},
		ParseExpansionLevel([]string{"charge", "customer"}))

	assert.Equal(t,
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{
			"charge": &ExpansionLevel{expansions: map[string]*ExpansionLevel{
				"customer": nil,
				"source":   nil,
			}},
			"customer": nil,
		}},
		ParseExpansionLevel([]string{"charge.customer", "customer", "charge.source"}))

	assert.Equal(t,
		&ExpansionLevel{expansions: map[string]*ExpansionLevel{
			"charge": &ExpansionLevel{expansions: map[string]*ExpansionLevel{
				"customer": &ExpansionLevel{expansions: map[string]*ExpansionLevel{
					"default_source": nil,
				}},
			}},
		}},
		ParseExpansionLevel([]string{"charge.customer.default_source", "charge"}))
}
