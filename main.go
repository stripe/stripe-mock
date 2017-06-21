//go:generate go-bindata openapi/fixtures.json openapi/spec2.json

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Fixtures struct {
	Resources map[string]interface{} `json:"resources"`
}

type OpenAPIDefinition struct {
	XResourceID string `json:"x-resourceId"`
}

type OpenAPIParameter struct {
	Description string                 `json:"description"`
	In          string                 `json:"in"`
	Name        string                 `json:"name"`
	Required    bool                   `json:"required"`
	Schema      map[string]interface{} `json:"schema"`
}

type OpenAPIMethod struct {
	Description string                                `json:"description"`
	OperationID string                                `json:"operation_id"`
	Parameters  []OpenAPIParameter                    `json:"parameters"`
	Responses   map[OpenAPIStatusCode]OpenAPIResponse `json:"responses"`
}

type OpenAPIPath string

type OpenAPIResponse struct {
	Description string `json:"description"`

	// Schema is the JSON schema of a response from the API.
	Schema map[string]interface{} `json:"schema"`
}

type OpenAPISpec struct {
	Definitions map[string]OpenAPIDefinition                  `json:"definitions"`
	Paths       map[OpenAPIPath]map[OpenAPIVerb]OpenAPIMethod `json:"paths"`
}

type OpenAPIStatusCode string

type OpenAPIVerb string

type StubServer struct {
	fixtures *Fixtures
	spec     *OpenAPISpec
}

func writeInternalError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "Internal server error")
}

func writeNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "Not found")
}

func (s *StubServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request: %v %v", r.Method, r.URL.Path)

	verbs, ok := s.spec.Paths[OpenAPIPath(r.URL.Path)]
	if !ok {
		writeNotFound(w)
		return
	}

	method, ok := verbs[OpenAPIVerb(strings.ToLower(r.Method))]
	if !ok {
		writeNotFound(w)
		return
	}

	response, ok := method.Responses["200"]
	if !ok {
		log.Printf("Couldn't find 200 response in spec")
		writeInternalError(w)
		return
	}

	log.Printf("schema %+v", response.Schema)

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

// ---

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
	log.Printf("parts %+v", parts)

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

// ---

func main() {
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

	log.Printf("Listening for API requests with %v method(s)",
		countAPIMethods(&spec))

	stub := StubServer{fixtures: &fixtures, spec: &spec}
	http.HandleFunc("/", stub.handleRequest)
	log.Fatal(http.ListenAndServe(":6065", nil))
}
