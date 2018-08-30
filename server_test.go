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

//
// Tests
//

func TestStubServer(t *testing.T) {
	resp, body := sendRequest(t, "POST", "/v1/charges",
		"amount=123", getDefaultHeaders())
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	assert.NoError(t, err)
	_, ok := data["id"]
	assert.True(t, ok)
}

func TestStubServer_MissingParam(t *testing.T) {
	resp, body := sendRequest(t, "POST", "/v1/charges",
		"", getDefaultHeaders())
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	assert.NoError(t, err)
	errorInfo, ok := data["error"].(map[string]interface{})
	assert.True(t, ok)
	errorType, ok := errorInfo["type"]
	assert.Equal(t, errorType, "invalid_request_error")
	assert.True(t, ok)
	message, ok := errorInfo["message"]
	assert.True(t, ok)
	assert.Contains(t, message, "object property 'amount' is required")
}

func TestStubServer_ExtraParam(t *testing.T) {
	resp, body := sendRequest(t, "POST", "/v1/charges",
		"amount=123&doesntexist=foo", getDefaultHeaders())
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	assert.NoError(t, err)
	errorInfo, ok := data["error"].(map[string]interface{})
	assert.True(t, ok)
	errorType, ok := errorInfo["type"]
	assert.Equal(t, errorType, "invalid_request_error")
	assert.True(t, ok)
	message, ok := errorInfo["message"]
	assert.True(t, ok)
	assert.Contains(t, message, "additional properties are not allowed: doesntexist")
}

func TestStubServer_QueryParam(t *testing.T) {
	resp, body := sendRequest(t, "GET", "/v1/charges?limit=10",
		"", getDefaultHeaders())
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	assert.NoError(t, err)
}

func TestStubServer_QueryExtraParam(t *testing.T) {
	resp, body := sendRequest(t, "GET", "/v1/charges?limit=10&doesntexist=foo",
		"", getDefaultHeaders())
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	assert.NoError(t, err)
	errorInfo, ok := data["error"].(map[string]interface{})
	assert.True(t, ok)
	errorType, ok := errorInfo["type"]
	assert.Equal(t, errorType, "invalid_request_error")
	assert.True(t, ok)
	message, ok := errorInfo["message"]
	assert.True(t, ok)
	assert.Contains(t, message, "additional properties are not allowed: doesntexist")
}

func TestStubServer_InvalidAuthorization(t *testing.T) {
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

func TestStubServer_AllowsContentTypeWithParameters(t *testing.T) {
	headers := getDefaultHeaders()
	headers["Content-Type"] = "application/x-www-form-urlencoded; charset=utf-8"

	resp, _ := sendRequest(t, "POST", "/v1/charges",
		"amount=123", headers)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
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
		"amount=123", headers)

	// Note the two spaces in front of "id" which indicate that our JSON is
	// pretty printed.
	assert.Contains(t, string(body), `  "id"`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestStubServer_ErrorsOnEmptyContentType(t *testing.T) {
	headers := getDefaultHeaders()
	headers["Content-Type"] = ""

	resp, body := sendRequest(t, "POST", "/v1/charges",
		"amount=123", headers)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	assert.NoError(t, err)
	errorInfo, ok := data["error"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "invalid_request_error", errorInfo["type"])
	assert.Equal(t,
		fmt.Sprintf(contentTypeEmpty, "application/x-www-form-urlencoded"),
		errorInfo["message"])
}

func TestStubServer_AllowsEmptyContentTypeOnDelete(t *testing.T) {
	headers := getDefaultHeaders()
	headers["Content-Type"] = ""

	resp, _ := sendRequest(t, "DELETE", "/v1/customers/cus_123", "", headers)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestStubServer_ErrorsOnMismatchedContentType(t *testing.T) {
	headers := getDefaultHeaders()
	headers["Content-Type"] = "application/json"

	resp, body := sendRequest(t, "POST", "/v1/charges",
		"amount=123", headers)
	fmt.Printf("body = %+v\n", string(body))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	assert.NoError(t, err)
	errorInfo, ok := data["error"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "invalid_request_error", errorInfo["type"])
	assert.Equal(t,
		fmt.Sprintf(contentTypeMismatched,
			"application/x-www-form-urlencoded",
			"application/json"),
		errorInfo["message"])
}

func TestStubServer_ReflectsIdempotencyKey(t *testing.T) {
	headers := getDefaultHeaders()
	headers["Idempotency-Key"] = "my-key"

	resp, _ := sendRequest(t, "POST", "/v1/charges",
		"amount=123", headers)
	assert.Equal(t, "my-key", resp.Header.Get("Idempotency-Key"))
}

func TestStubServer_RoutesRequest(t *testing.T) {
	server := getStubServer(t)

	{
		route, pathParams := server.routeRequest(
			&http.Request{Method: "GET", URL: &url.URL{Path: "/v1/charges"}})
		assert.NotNil(t, route)
		assert.Equal(t, chargeAllMethod, route.operation)
		assert.Nil(t, pathParams)
	}

	{
		route, pathParams := server.routeRequest(
			&http.Request{Method: "POST", URL: &url.URL{Path: "/v1/charges"}})
		assert.NotNil(t, route)
		assert.Equal(t, chargeCreateMethod, route.operation)
		assert.Nil(t, pathParams)
	}

	{
		route, pathParams := server.routeRequest(
			&http.Request{Method: "GET", URL: &url.URL{Path: "/v1/charges/ch_123"}})
		assert.NotNil(t, route)
		assert.Equal(t, chargeGetMethod, route.operation)
		assert.Equal(t, "ch_123", *(*pathParams).PrimaryID)
		assert.Equal(t, []*PathParamsSecondaryID(nil), (*pathParams).SecondaryIDs)
	}

	{
		route, pathParams := server.routeRequest(
			&http.Request{Method: "DELETE", URL: &url.URL{Path: "/v1/customers/cus_123"}})
		assert.NotNil(t, route)
		assert.Equal(t, customerDeleteMethod, route.operation)
		assert.Equal(t, "cus_123", *(*pathParams).PrimaryID)
		assert.Equal(t, []*PathParamsSecondaryID(nil), (*pathParams).SecondaryIDs)
	}

	{
		route, pathParams := server.routeRequest(
			&http.Request{Method: "GET", URL: &url.URL{Path: "/v1/doesnt-exist"}})
		assert.Equal(t, (*stubServerRoute)(nil), route)
		assert.Nil(t, pathParams)
	}

	// Route with a parameter, but not an object's primary ID
	{
		route, pathParams := server.routeRequest(
			&http.Request{Method: "POST",
				URL: &url.URL{Path: "/v1/invoices/in_123/pay"}})
		assert.NotNil(t, route)
		assert.Equal(t, invoicePayMethod, route.operation)
		assert.Equal(t, "in_123", *(*pathParams).PrimaryID)
		assert.Equal(t, []*PathParamsSecondaryID(nil), (*pathParams).SecondaryIDs)
	}

	// Route with a parameter, but not an object's primary ID
	{
		route, pathParams := server.routeRequest(
			&http.Request{Method: "GET",
				URL: &url.URL{Path: "/v1/application_fees/fee_123/refunds"}})
		assert.NotNil(t, route)
		assert.Equal(t, invoicePayMethod, route.operation)
		assert.Equal(t, (*string)(nil), (*pathParams).PrimaryID)
		assert.Equal(t, 1, len((*pathParams).SecondaryIDs))
		assert.Equal(t, "fee_123", (*pathParams).SecondaryIDs[0].ID)
		assert.Equal(t, "fee", (*pathParams).SecondaryIDs[0].Name)
	}

	// Route with multiple parameters in its URL
	{
		route, pathParams := server.routeRequest(
			&http.Request{Method: "GET",
				URL: &url.URL{Path: "/v1/application_fees/fee_123/refunds/fr_123"}})
		assert.NotNil(t, route)
		assert.Equal(t, applicationFeeRefundGetMethod, route.operation)
		assert.Equal(t, "fr_123", *(*pathParams).PrimaryID)
		assert.Equal(t, 1, len((*pathParams).SecondaryIDs))
		assert.Equal(t, "fee_123", (*pathParams).SecondaryIDs[0].ID)
		assert.Equal(t, "fee", (*pathParams).SecondaryIDs[0].Name)
	}
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
	mediaType, schema := getRequestBodySchema(operation)
	assert.NotNil(t, mediaType)
	assert.Equal(t, "application/x-www-form-urlencoded", *mediaType)
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
	mediaType, schema := getRequestBodySchema(method)
	assert.Nil(t, mediaType)
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
// Tests for private functions
//

func TestCompilePath(t *testing.T) {
	{
		pattern, pathParamNames := compilePath(spec.Path("/v1/charges"))
		assert.Equal(t, `\A/v1/charges\z`, pattern.String())
		assert.Equal(t, []string(nil), pathParamNames)
	}

	{
		pattern, pathParamNames := compilePath(spec.Path("/v1/charges/{id}"))
		assert.Equal(t, `\A/v1/charges/(?P<id>[\w-_.]+)\z`, pattern.String())
		assert.Equal(t, []string{"id"}, pathParamNames)
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
			assert.Equal(t, tc.want, parseExpansionLevel(tc.expansions))
		})
	}
}

//
// Private functions
//

func encode64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func getDefaultHeaders() map[string]string {
	headers := make(map[string]string)
	headers["Authorization"] = "Bearer sk_test_123"
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	return headers
}

func getStubServer(t *testing.T) *StubServer {
	server := &StubServer{spec: &testSpec, fixtures: &testFixtures}
	err := server.initializeRouter()
	assert.NoError(t, err)
	return server
}

func sendRequest(t *testing.T, method string, url string, params string,
	headers map[string]string) (*http.Response, []byte) {

	server := getStubServer(t)

	fullURL := fmt.Sprintf("https://stripe.com%s", url)
	req := httptest.NewRequest(method, fullURL, bytes.NewBufferString(params))
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
