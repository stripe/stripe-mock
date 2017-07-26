package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/brandur/stripe-mock/param/coercer"
	"github.com/brandur/stripe-mock/param/parser"
	"github.com/brandur/stripe-mock/spec"
	"github.com/lestrrat/go-jsschema"
	"github.com/lestrrat/go-jsval"
	"github.com/lestrrat/go-jsval/builder"
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
	pattern   *regexp.Regexp
	method    *spec.Method
	validator *jsval.JSVal
}

// HandleRequest handes an HTTP request directed at the API stub.
func (s *StubServer) HandleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("Request: %v %v", r.Method, r.URL.Path)

	route := s.routeRequest(r)
	if route == nil {
		writeResponse(w, start, http.StatusNotFound, nil)
		return
	}

	response, ok := route.method.Responses["200"]
	if !ok {
		log.Printf("Couldn't find 200 response in spec")
		writeResponse(w, start, http.StatusInternalServerError, nil)
		return
	}

	if verbose {
		log.Printf("Response schema: %+v", response.Schema)
	}

	var formString string
	if r.Method == "GET" {
		formString = r.URL.RawQuery
	} else {
		formBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Couldn't read request body: %v", err)
			writeResponse(w, start, http.StatusInternalServerError, nil)
			return
		}
		r.Body.Close()
		formString = string(formBytes)
	}
	requestData, err := parser.ParseFormString(formString)
	if err != nil {
		log.Printf("Couldn't parse query/body: %v", err)
		writeResponse(w, start, http.StatusInternalServerError, nil)
		return
	}

	if verbose {
		log.Printf("Request data: %+v", requestData)
	}

	// OpenAPI 2.0 stores a possible JSON schema in a special parameter that's
	// identifiable with "in: body". Currently I'm only doing parameter
	// validation if we have one of these (which is usually POST verbs).
	// Everything goes to JSON Schema for OpenAPI 3.0, so we'll be able to
	// support validation all verbs, and much more simply.
	requestSchema := bodyParameterSchema(route.method)
	if requestSchema != nil {
		coercer.CoerceParams(requestSchema, requestData)

		err := route.validator.Validate(requestData)
		if err != nil {
			log.Printf("Validation error: %v", err)
			responseData := fmt.Sprintf("Request error: %v", err)
			writeResponse(w, start, http.StatusBadRequest, responseData)
			return
		}
	}

	expansions, rawExpansions := extractExpansions(requestData)
	if verbose {
		log.Printf("Expansions: %+v", rawExpansions)
	}

	generator := DataGenerator{s.spec.Definitions, s.fixtures}
	responseData, err := generator.Generate(response.Schema, r.URL.Path, expansions)
	if err != nil {
		log.Printf("Couldn't generate response: %v", err)
		writeResponse(w, start, http.StatusInternalServerError, nil)
		return
	}
	writeResponse(w, start, http.StatusOK, responseData)
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
			log.Printf("Compiled path: %v", pathPattern.String())
		}

		for verb, method := range verbs {
			numEndpoints++

			validator, err := getValidator(method)
			if err != nil {
				return err
			}

			// Note that this may be nil if no suitable validator could be
			// generated.
			if validator != nil {
				numValidators++
			}

			route := stubServerRoute{
				pattern:   pathPattern,
				method:    method,
				validator: validator,
			}

			// net/http will always give us verbs in uppercase, so build our
			// routing table this way too
			verb = spec.HTTPVerb(strings.ToUpper(string(verb)))

			s.routes[verb] = append(s.routes[verb], route)
		}
	}

	log.Printf("Routing to %v path(s) and %v endpoint(s) with %v validator(s)",
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

func bodyParameterSchema(method *spec.Method) *spec.JSONSchema {
	for _, param := range method.Parameters {
		if param.In == "body" {
			return param.Schema
		}
	}
	return nil
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

func getValidator(method *spec.Method) (*jsval.JSVal, error) {
	for _, parameter := range method.Parameters {
		if parameter.Schema != nil {
			schema := schema.New()
			err := schema.Extract(parameter.Schema.RawFields)
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
	}
	return nil, nil
}

func writeResponse(w http.ResponseWriter, start time.Time, status int, data interface{}) {
	if data == nil {
		data = []byte(http.StatusText(status))
	}

	encodedData, err := json.Marshal(&data)
	if err != nil {
		log.Printf("Error serializing response: %v", err)
		writeResponse(w, start, http.StatusInternalServerError, nil)
		return
	}

	w.Header().Set("Stripe-Mock-Version", version)

	w.WriteHeader(status)
	_, err = w.Write(encodedData)
	if err != nil {
		log.Printf("Error writing to client: %v", err)
	}
	log.Printf("Response: elapsed=%v status=%v", time.Now().Sub(start), status)
}
