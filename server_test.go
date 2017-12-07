package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/stripe/stripe-mock/spec"
)

func TestStubServer(t *testing.T) {
	resp, body := sendRequest(t, "POST", "/v1/charges",
		"amount=123&currency=usd", getDefaultHeaders())
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	assert.NoError(t, err)
	_, ok := data["id"]
	assert.True(t, ok)
}

func TestStubServerError(t *testing.T) {
	resp, body := sendRequest(t, "GET", "/a", "", nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	assert.NoError(t, err)
	errorInfo, ok := data["error"].(map[string]interface{})
	assert.True(t, ok)
	errorType, ok := errorInfo["type"]
	assert.Equal(t, errorType, "invalid_request_error")
	assert.True(t, ok)
	_, ok = errorInfo["message"]
	assert.True(t, ok)
}

func TestStubServer_SetsSpecialHeaders(t *testing.T) {
	resp, _ := sendRequest(t, "POST", "/", "", nil)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, version, resp.Header.Get("Stripe-Mock-Version"))
	_, ok := resp.Header["Request-Id"]
	assert.False(t, ok)

	resp, _ = sendRequest(t, "POST", "/", "", getDefaultHeaders())
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, version, resp.Header.Get("Stripe-Mock-Version"))
	assert.Equal(t, "req_123", resp.Header.Get("Request-Id"))
}

func TestStubServer_ParameterValidation(t *testing.T) {
	resp, body := sendRequest(t, "POST", "/v1/charges", "", getDefaultHeaders())
	assert.Contains(t, string(body), "property 'amount' is required")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestStubServer_FormatsForCurl(t *testing.T) {
	headers := getDefaultHeaders()
	headers["User-Agent"] = "curl/1.2.3"
	resp, body := sendRequest(t, "POST", "/v1/charges",
		"amount=123&currency=usd", headers)

	// Note the two spaces in front of "id" which indicate that our JSON is
	// pretty printed.
	assert.Contains(t, string(body), `  "id"`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestStubServer_RoutesRequest(t *testing.T) {
	server := getStubServer(t)
	var route *stubServerRoute

	route = server.routeRequest(
		&http.Request{Method: "GET", URL: &url.URL{Path: "/v1/charges"}})
	assert.NotNil(t, route)
	assert.Equal(t, chargeAllMethod, route.operation)

	route = server.routeRequest(
		&http.Request{Method: "POST", URL: &url.URL{Path: "/v1/charges"}})
	assert.NotNil(t, route)
	assert.Equal(t, chargeCreateMethod, route.operation)

	route = server.routeRequest(
		&http.Request{Method: "GET", URL: &url.URL{Path: "/v1/charges/ch_123"}})
	assert.NotNil(t, route)
	assert.Equal(t, chargeGetMethod, route.operation)

	route = server.routeRequest(
		&http.Request{Method: "DELETE", URL: &url.URL{Path: "/v1/charges/ch_123"}})
	assert.NotNil(t, route)
	assert.Equal(t, chargeDeleteMethod, route.operation)

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
	operation := &spec.Operation{RequestBody: &spec.RequestBody{
		Content: map[string]spec.MediaType{
			"application/x-www-form-urlencoded": {
				Schema: &spec.Schema{
					Properties: map[string]*spec.Schema{
						"name": {
							Type: "string",
						},
					},
				},
			},
		},
	}}
	schema := getRequestBodySchema(operation)
	assert.NotNil(t, schema)
	validator, err := spec.GetValidatorForOpenAPI3Schema(schema, nil)
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
	method := &spec.Operation{Parameters: []*spec.Parameter{
		{Schema: nil},
	}}
	schema := getRequestBodySchema(method)
	assert.Nil(t, schema)
}

func TestIsCurl(t *testing.T) {
	testCases := []struct {
		userAgent string
		want      bool
	}{
		{"curl/7.51.0", true},

		// false because it's not something (to my knowledge) that cURL would
		// ever return
		{"curl", false},

		{"Mozilla", false},
		{"", false},
	}
	for _, tc := range testCases {
		t.Run(tc.userAgent, func(t *testing.T) {
			assert.Equal(t, tc.want, isCurl(tc.userAgent))
		})
	}
}

func TestParseExpansionLevel(t *testing.T) {
	emptyExpansionLevel := &ExpansionLevel{
		expansions: make(map[string]*ExpansionLevel),
	}
	testCases := []struct {
		expansions []string
		want       *ExpansionLevel
	}{
		{
			[]string{"charge", "customer"},
			&ExpansionLevel{expansions: map[string]*ExpansionLevel{
				"charge":   emptyExpansionLevel,
				"customer": emptyExpansionLevel,
			}},
		},
		{
			[]string{"charge.customer", "customer", "charge.source"},
			&ExpansionLevel{expansions: map[string]*ExpansionLevel{
				"charge": {expansions: map[string]*ExpansionLevel{
					"customer": emptyExpansionLevel,
					"source":   emptyExpansionLevel,
				}},
				"customer": emptyExpansionLevel,
			}},
		},
		{
			[]string{"charge.customer.default_source", "charge"},
			&ExpansionLevel{expansions: map[string]*ExpansionLevel{
				"charge": {expansions: map[string]*ExpansionLevel{
					"customer": {expansions: map[string]*ExpansionLevel{
						"default_source": emptyExpansionLevel,
					}},
				}},
			}},
		},
		{
			[]string{"*"},
			&ExpansionLevel{
				expansions: map[string]*ExpansionLevel{},
				wildcard:   true,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%+v", tc.expansions), func(t *testing.T) {
			assert.Equal(t, tc.want, ParseExpansionLevel(tc.expansions))
		})
	}
}

func TestValidateAuth(t *testing.T) {
	testCases := []struct {
		auth string
		want bool
	}{
		{"Basic " + encode64("sk_test_123"), true},
		{"Bearer sk_test_123", true},
		{"", false},
		{"Bearer", false},
		{"Basic", false},
		{"Bearer ", false},
		{"Basic ", false},
		{"Basic 123", false}, // "123" is not a valid key when base64 decoded
		{"Basic " + encode64("sk_test"), false},
		{"Bearer sk_test_123 extra", false},
		{"Bearer sk_test", false},
		{"Bearer sk_test_123_extra", false},
		{"Bearer sk_live_123", false},
		{"Bearer sk_test_", false},
	}
	for _, tc := range testCases {
		t.Run("Authorization: "+tc.auth, func(t *testing.T) {
			assert.Equal(t, tc.want, validateAuth(tc.auth))
		})
	}
}

//
// ---
//

func encode64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func getStubServer(t *testing.T) *StubServer {
	server := &StubServer{spec: &testSpec, fixtures: &testFixtures}
	err := server.initializeRouter()
	assert.NoError(t, err)
	return server
}

func getDefaultHeaders() map[string]string {
	headers := make(map[string]string)
	headers["Authorization"] = "Bearer sk_test_123"
	return headers
}

func sendRequest(t *testing.T, method string, url string, params string,
	headers map[string]string) (*http.Response, []byte) {
	server := getStubServer(t)

	fullUrl := fmt.Sprintf("https://stripe.com%s", url)
	req := httptest.NewRequest(method, fullUrl, bytes.NewBufferString(params))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	server.HandleRequest(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	return resp, body
}
