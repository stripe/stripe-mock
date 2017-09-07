package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/lestrrat/go-jsschema"
	"github.com/lestrrat/go-jsval"
	"github.com/lestrrat/go-jsval/builder"
	"github.com/stripe/stripe-mock/param/coercer"
	"github.com/stripe/stripe-mock/param/parser"
	"github.com/stripe/stripe-mock/spec"
)

const (
	invalidAuthorization = "Please authenticate by specifying an " +
		"`Authorization` header with any valid looking testmode secret API " +
		"key. For example, `Authorization: Bearer sk_test_123`. " +
		"Authorization was '%s'."
)

// ExpansionLevel represents expansions on a single "level" of resource. It may
// have subexpansions that are meant to take effect on resources that are
// nested below it (on other levels).
type ExpansionLevel struct {
	expansions map[string]*ExpansionLevel

	// wildcard specifies that everything should be expanded.
	wildcard bool
}

// ParseExpansionLevel parses a set of raw expansions from a request query
// string or form and produces a structure more useful for performing actual
// expansions.
func ParseExpansionLevel(raw []string) *ExpansionLevel {
	sort.Strings(raw)

	level := &ExpansionLevel{expansions: make(map[string]*ExpansionLevel)}
	groups := make(map[string][]string)

	for _, expansion := range raw {
		parts := strings.Split(expansion, ".")
		if len(parts) == 1 {
			if parts[0] == "*" {
				level.wildcard = true
			} else {
				level.expansions[parts[0]] = nil
			}
		} else {
			groups[parts[0]] = append(groups[parts[0]], strings.Join(parts[1:], "."))
		}
	}

	for key, subexpansions := range groups {
		level.expansions[key] = ParseExpansionLevel(subexpansions)
	}

	return level
}

// StubServer handles incoming HTTP requests and responds to them appropriately
// based off the set of OpenAPI routes that it's been configured with.
type StubServer struct {
	fixtures *spec.Fixtures
	routes   map[spec.HTTPVerb][]stubServerRoute
	spec     *spec.Spec
}

// stubServerRoute is a single route in a StubServer's routing table. It has a
// pattern to match an incoming path and a description of the method that would
// be executed in the event of a match.
type stubServerRoute struct {
	pattern              *regexp.Regexp
	operation            *spec.Operation
	requestBodyValidator *jsval.JSVal
}

// HandleRequest handes an HTTP request directed at the API stub.
func (s *StubServer) HandleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	fmt.Printf("Request: %v %v\n", r.Method, r.URL.Path)

	auth := r.Header.Get("Authorization")
	if !validateAuth(auth) {
		writeResponse(w, r, start, http.StatusUnauthorized,
			fmt.Sprintf(invalidAuthorization, auth))
		return
	}

	route := s.routeRequest(r)
	if route == nil {
		writeResponse(w, r, start, http.StatusNotFound, nil)
		return
	}

	response, ok := route.operation.Responses["200"]
	if !ok {
		fmt.Printf("Couldn't find 200 response in spec\n")
		writeResponse(w, r, start, http.StatusInternalServerError, nil)
		return
	}

	if verbose {
		fmt.Printf("Response: %+v\n", response)
		schema := response.Content["application/x-www-form-urlencoded"]
		fmt.Printf("Response schema: %+v\n", schema)
		fmt.Printf("Response schema ref: '%+v'\n", schema.Ref)
	}

	var formString string
	if r.Method == "GET" {
		formString = r.URL.RawQuery
	} else {
		formBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("Couldn't read request body: %v\n", err)
			writeResponse(w, r, start, http.StatusInternalServerError, nil)
			return
		}
		r.Body.Close()
		formString = string(formBytes)
	}
	requestData, err := parser.ParseFormString(formString)
	if err != nil {
		fmt.Printf("Couldn't parse query/body: %v\n", err)
		writeResponse(w, r, start, http.StatusInternalServerError, nil)
		return
	}

	if verbose {
		fmt.Printf("Request data: %+v\n", requestData)
	}

	// Currently we only validate parameters in the request body, but we should
	// really validate query and URL parameters as well now that we've
	// transitioned to OpenAPI 3.0
	bodySchema := getRequestBodySchema(route.operation)
	if bodySchema != nil {
		coercer.CoerceParams(bodySchema, requestData)

		err := route.requestBodyValidator.Validate(requestData)
		if err != nil {
			fmt.Printf("Validation error: %v\n", err)
			responseData := fmt.Sprintf("Request error: %v", err)
			writeResponse(w, r, start, http.StatusBadRequest, responseData)
			return
		}
	}

	expansions, rawExpansions := extractExpansions(requestData)
	if verbose {
		fmt.Printf("Expansions: %+v\n", rawExpansions)
	}

	generator := DataGenerator{s.spec.Components.Schemas, s.fixtures}
	responseData, err := generator.Generate(
		response.Content["application/x-www-form-urlencoded"],
		r.URL.Path,
		expansions)
	if err != nil {
		fmt.Printf("Couldn't generate response: %v\n", err)
		writeResponse(w, r, start, http.StatusInternalServerError, nil)
		return
	}
	if verbose {
		fmt.Printf("Response data: %+v\n", responseData)
	}
	writeResponse(w, r, start, http.StatusOK, responseData)
}

func (s *StubServer) initializeRouter() error {
	var numEndpoints int
	var numPaths int
	var numValidators int

	s.routes = make(map[spec.HTTPVerb][]stubServerRoute)

	for path, verbs := range s.spec.Paths {
		numPaths++

		pathPattern := compilePath(path)

		if verbose {
			fmt.Printf("Compiled path: %v\n", pathPattern.String())
		}

		for verb, operation := range verbs {
			numEndpoints++

			requestBodyValidator, err := getRequestBodyValidator(operation)
			if err != nil {
				return err
			}

			// Note that this may be nil if no suitable validator could be
			// generated.
			if requestBodyValidator != nil {
				numValidators++
			}

			route := stubServerRoute{
				pattern:              pathPattern,
				operation:            operation,
				requestBodyValidator: requestBodyValidator,
			}

			// net/http will always give us verbs in uppercase, so build our
			// routing table this way too
			verb = spec.HTTPVerb(strings.ToUpper(string(verb)))

			s.routes[verb] = append(s.routes[verb], route)
		}
	}

	fmt.Printf("Routing to %v path(s) and %v endpoint(s) with %v validator(s)\n",
		numPaths, numEndpoints, numValidators)
	return nil
}

func (s *StubServer) routeRequest(r *http.Request) *stubServerRoute {
	verbRoutes := s.routes[spec.HTTPVerb(r.Method)]
	for _, route := range verbRoutes {
		if route.pattern.MatchString(r.URL.Path) {
			return &route
		}
	}
	return nil
}

// ---

func getRequestBodySchema(operation *spec.Operation) *spec.JSONSchema {
	fmt.Printf("Operation: %+v\n", operation)
	if operation.RequestBody == nil {
		return nil
	}
	mediaType, mediaTypePresent :=
		operation.RequestBody.Content["application/x-www-form-urlencoded"]
	fmt.Printf("mediaType: %+v mediaTypePresent: %+v\n", mediaType, mediaTypePresent)
	if !mediaTypePresent {
		return nil
	}
	return mediaType.Schema
}

var pathParameterPattern = regexp.MustCompile(`\{(\w+)\}`)

func compilePath(path spec.Path) *regexp.Regexp {
	pattern := `\A`
	parts := strings.Split(string(path), "/")

	for _, part := range parts {
		if part == "" {
			continue
		}

		submatches := pathParameterPattern.FindAllStringSubmatch(part, -1)
		if submatches == nil {
			pattern += `/` + part
		} else {
			pattern += `/(?P<` + submatches[0][1] + `>[\w-_.]+)`
		}
	}

	return regexp.MustCompile(pattern + `\z`)
}

func extractExpansions(data map[string]interface{}) (*ExpansionLevel, []string) {
	expand, ok := data["expand"]
	if !ok {
		return nil, nil
	}

	var expansions []string

	expandStr, ok := expand.(string)
	if ok {
		expansions = append(expansions, expandStr)
		return ParseExpansionLevel(expansions), expansions
	}

	expandArr, ok := expand.([]interface{})
	if ok {
		for _, expand := range expandArr {
			expandStr := expand.(string)
			expansions = append(expansions, expandStr)
		}
		return ParseExpansionLevel(expansions), expansions
	}

	return nil, nil
}

func getRequestBodyValidator(operation *spec.Operation) (*jsval.JSVal, error) {
	requestBodySchema := getRequestBodySchema(operation)
	if requestBodySchema == nil {
		return nil, nil
	}

	schema := schema.New()
	err := schema.Extract(requestBodySchema.RawFields)
	if err != nil {
		return nil, err
	}

	validatorBuilder := builder.New()
	validator, err := validatorBuilder.Build(schema)
	if err != nil {
		return nil, err
	}

	return validator, nil
}

func isCurl(userAgent string) bool {
	return strings.HasPrefix(userAgent, "curl/")
}

func writeResponse(w http.ResponseWriter, r *http.Request, start time.Time, status int, data interface{}) {
	if data == nil {
		data = http.StatusText(status)
	}

	var encodedData []byte
	var err error

	if isCurl(r.Header.Get("User-Agent")) {
		encodedData, err = json.Marshal(&data)
	} else {
		encodedData, err = json.MarshalIndent(&data, "", "  ")
	}

	if err != nil {
		fmt.Printf("Error serializing response: %v\n", err)
		writeResponse(w, r, start, http.StatusInternalServerError, nil)
		return
	}

	w.Header().Set("Stripe-Mock-Version", version)

	w.WriteHeader(status)
	_, err = w.Write(encodedData)
	if err != nil {
		fmt.Printf("Error writing to client: %v\n", err)
	}
	fmt.Printf("Response: elapsed=%v status=%v\n", time.Now().Sub(start), status)
}

func validateAuth(auth string) bool {
	if auth == "" {
		return false
	}

	parts := strings.Split(auth, " ")

	// Expect ["Bearer", "sk_test_123"] or ["Basic", "aaaaa"]
	if len(parts) != 2 || parts[1] == "" {
		return false
	}

	var key string
	switch parts[0] {
	case "Basic":
		keyBytes, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return false
		}
		key = string(keyBytes)

	case "Bearer":
		key = parts[1]

	default:
		return false
	}

	keyParts := strings.Split(key, "_")

	// Expect ["sk", "test", "123"]
	if len(keyParts) != 3 {
		return false
	}

	if keyParts[0] != "sk" {
		return false
	}

	if keyParts[1] != "test" {
		return false
	}

	// Expect something (anything but an empty string) in the third position
	if len(keyParts[2]) == 0 {
		return false
	}

	return true
}
