//go:generate go-bindata openapi/fixtures.json openapi/spec2.json

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type Fixtures struct {
	Resources map[string]interface{} `json:"resources"`
}

type HTTPVerb string

type JSONSchema map[string]interface{}

type OpenAPIDefinition struct {
	XResourceID string `json:"x-resourceId"`
}

type OpenAPIParameter struct {
	Description string     `json:"description"`
	In          string     `json:"in"`
	Name        string     `json:"name"`
	Required    bool       `json:"required"`
	Schema      JSONSchema `json:"schema"`
}

type OpenAPIMethod struct {
	Description string                                `json:"description"`
	OperationID string                                `json:"operation_id"`
	Parameters  []OpenAPIParameter                    `json:"parameters"`
	Responses   map[OpenAPIStatusCode]OpenAPIResponse `json:"responses"`
}

type OpenAPIPath string

type OpenAPIResponse struct {
	Description string     `json:"description"`
	Schema      JSONSchema `json:"schema"`
}

type OpenAPISpec struct {
	Definitions map[string]OpenAPIDefinition                `json:"definitions"`
	Paths       map[OpenAPIPath]map[HTTPVerb]*OpenAPIMethod `json:"paths"`
}

type OpenAPIStatusCode string

type StubServerRoute struct {
	pattern *regexp.Regexp
	method  *OpenAPIMethod
}

type StubServer struct {
	fixtures *Fixtures
	routes   map[HTTPVerb][]StubServerRoute
	spec     *OpenAPISpec
}

func (s *StubServer) routeRequest(r *http.Request) *OpenAPIMethod {
	verbRoutes := s.routes[HTTPVerb(r.Method)]
	for _, route := range verbRoutes {
		if route.pattern.MatchString(r.URL.Path) {
			return route.method
		}
	}
	return nil
}

func (s *StubServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request: %v %v", r.Method, r.URL.Path)
	start := time.Now()
	defer func() {
		log.Printf("Response: elapsed=%v status=200", time.Now().Sub(start))
	}()

	method := s.routeRequest(r)
	if method == nil {
		writeNotFound(w)
		return
	}

	response, ok := method.Responses["200"]
	if !ok {
		log.Printf("Couldn't find 200 response in spec")
		writeInternalError(w)
		return
	}

	if Verbose {
		log.Printf("Response schema: %+v", response.Schema)
	}

	ref, ok := response.Schema["$ref"].(string)
	if !ok {
		log.Printf("Expected response to include $ref")
		writeInternalError(w)
		return
	}

	definition, err := definitionFromJSONPointer(ref)
	if err != nil {
		log.Printf("Error extracting definition: %v", err)
		writeInternalError(w)
		return
	}

	resource, ok := s.spec.Definitions[definition]
	if !ok {
		log.Printf("Expected definitions to include %v", ref)
		writeInternalError(w)
		return
	}

	fixture, ok := s.fixtures.Resources[resource.XResourceID]
	if !ok {
		log.Printf("Expected fixtures to include %v", resource.XResourceID)
		writeInternalError(w)
		return
	}

	data, err := json.Marshal(&fixture)
	if err != nil {
		log.Printf("Error serializing fixture: %v", err)
		writeInternalError(w)
		return
	}

	w.Write(data)
}

func (s *StubServer) initializeRouter() {
	var numEndpoints int
	var numPaths int

	s.routes = make(map[HTTPVerb][]StubServerRoute)

	for path, verbs := range s.spec.Paths {
		numPaths += 1

		pathPattern := compilePath(path)

		if Verbose {
			log.Printf("Compiled path: %v", pathPattern.String())
		}

		for verb, method := range verbs {
			numEndpoints += 1

			route := StubServerRoute{
				pattern: pathPattern,
				method:  method,
			}

			// net/http will always give us verbs in uppercase, so build our
			// routing table this way too
			verb = HTTPVerb(strings.ToUpper(string(verb)))

			s.routes[verb] = append(s.routes[verb], route)
		}
	}

	log.Printf("Routing to %v path(s) and %v endpoint(s)",
		numPaths, numEndpoints)
}

// ---

var pathParameterPattern = regexp.MustCompile(`\{(\w+)\}`)

func compilePath(path OpenAPIPath) *regexp.Regexp {
	var pattern string
	parts := strings.Split(string(path), "/")

	for _, part := range parts {
		if part == "" {
			continue
		}

		submatches := pathParameterPattern.FindAllStringSubmatch(part, -1)
		if submatches == nil {
			pattern += `/` + part
		} else {
			pattern += `/(?P<` + submatches[0][1] + `>\w+)`
		}
	}

	return regexp.MustCompile(pattern)
}

// countAPIMethods counts the number of separate API methods that the spec is
// handling. That's all verbs across all paths.
func countAPIMethods(spec *OpenAPISpec) int {
	count := 0
	for _, verbs := range spec.Paths {
		count += len(verbs)
	}
	return count
}

// definitionFromJSONPointer extracts the name of a JSON schema definition from
// a JSON pointer, so "#/definitions/charge" would become just "charge". This
// is a simplified workaround to avoid bringing in JSON schema infrastructure
// because we can guarantee that the spec we're producing will take a certain
// shape. If this gets too hacky, it will be better to put a more legitimate
// JSON schema parser in place.
func definitionFromJSONPointer(pointer string) (string, error) {
	parts := strings.Split(pointer, "/")

	if parts[0] != "#" {
		return "", fmt.Errorf("Expected '#' in 0th part of pointer %v", pointer)
	}

	if parts[1] != "definitions" {
		return "", fmt.Errorf("Expected 'definitions' in 1st part of pointer %v",
			pointer)
	}

	if len(parts) > 3 {
		return "", fmt.Errorf("Pointer too long to be handle %v", pointer)
	}

	return parts[2], nil
}

func writeInternalError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "Internal server error")
}

func writeNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "Not found")
}

// ---

var Verbose bool

func main() {
	if os.Getenv("STRIPE_VERBOSE") == "true" {
		Verbose = true
	}

	// Load the spec information from go-bindata
	data, err := Asset("openapi/spec2.json")
	if err != nil {
		log.Fatalf("Error loading spec: %v", err)
	}

	var spec OpenAPISpec
	err = json.Unmarshal(data, &spec)
	if err != nil {
		log.Fatalf("Error decoding spec: %v", err)
	}

	// And do the same for fixtures
	data, err = Asset("openapi/fixtures.json")
	if err != nil {
		log.Fatalf("Error loading fixtures: %v", err)
	}

	var fixtures Fixtures
	err = json.Unmarshal(data, &fixtures)
	if err != nil {
		log.Fatalf("Error decoding spec: %v", err)
	}

	stub := StubServer{fixtures: &fixtures, spec: &spec}
	stub.initializeRouter()

	http.HandleFunc("/", stub.handleRequest)
	log.Printf("Listening on :6065")
	log.Fatal(http.ListenAndServe(":6065", nil))
}
