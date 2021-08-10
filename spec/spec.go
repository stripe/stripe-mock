package spec

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

//
// Public values
//

// A set of constants for the different types of possible OpenAPI parameters.
const (
	ParameterPath  = "path"
	ParameterQuery = "query"
)

// A set of constant for the named types available in JSON Schema.
const (
	TypeArray   = "array"
	TypeBoolean = "boolean"
	TypeInteger = "integer"
	TypeNumber  = "number"
	TypeObject  = "object"
	TypeString  = "string"
)

//
// Public types
//

// Components is a struct for the components section of an OpenAPI
// specification.
type Components struct {
	Schemas map[string]*Schema `json:"schemas"`
}

// ExpansionResources is a struct for possible expansions in a resource.
type ExpansionResources struct {
	OneOf []*Schema `json:"oneOf"`
}

// Fixtures is a struct for a set of companion fixtures for an OpenAPI
// specification.
type Fixtures struct {
	Resources map[ResourceID]interface{} `json:"resources"`
}

// isJSONFile judges based on a file's extension whether it's a JSON file. It's
// used to return a better error message if the user points to an unsupported
// file.
func isJSONFile(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".json"
}

//LoadFixtures attempts to loads the given embedded fixture.
func LoadFixtures(fixturesPath string) (*Fixtures, error) {
	var data []byte
	var err error

	if fixturesPath == "" {
		// And do the same for fixtures
		data, err = Asset("openapi/openapi/fixtures3.json")
	} else {
		if !isJSONFile(fixturesPath) {
			return nil, fmt.Errorf("Fixtures should come from a JSON file")
		}

		data, err = ioutil.ReadFile(fixturesPath)
	}

	if err != nil {
		return nil, fmt.Errorf("error loading fixtures: %v", err)
	}

	var fixtures Fixtures
	err = json.Unmarshal(data, &fixtures)
	if err != nil {
		return nil, fmt.Errorf("error decoding spec: %v", err)
	}

	return &fixtures, nil
}

// HTTPVerb is a type for an HTTP verb like GET, POST, etc.
type HTTPVerb string

// Info is the `info` portion of an OpenAPI specification that contains meta
// information about it.
type Info struct {
	// Version is the Stripe API version represented in the specification. It
	// takes a date-based form like `2019-02-19`.
	Version string `json:"version"`
}

// This is a list of fields that either we handle properly or we're confident
// it's safe to ignore. If a field not in this list appears in the OpenAPI spec,
// then we'll get an error so we remember to update stripe-mock to support it.
var supportedSchemaFields = []string{
	"$ref",
	"additionalProperties",
	"anyOf",
	"description",
	"enum",
	"format",
	"items",
	"maxLength",
	"nullable",
	"pattern",
	"properties",
	"required",
	"title",
	"type",
	"x-expandableFields",
	"x-expansionResources",
	"x-resourceId",

	// This is currently being used to store additional metadata for our SDKs. It's
	// passed through our Spec and should be ignored
	"x-stripeResource",

	// This is currently a hint for the server-side so I haven't included it in
	// Schema yet. If we do start validating responses that come out of
	// stripe-mock, we may need to observe this as well.
	"x-stripeBypassValidation",
}

// Schema is a struct representing a JSON schema.
type Schema struct {
	// AdditionalProperties is either a `false` to indicate that no additional
	// properties in the object are allowed (beyond what's in Properties), or a
	// JSON schema that describes the expected format of any additional properties.
	//
	// We currently just read it as an `interface{}` because we're not using it
	// for anything right now.
	AdditionalProperties interface{} `json:"additionalProperties,omitempty"`

	AnyOf      []*Schema          `json:"anyOf,omitempty"`
	Enum       []interface{}      `json:"enum,omitempty"`
	Format     string             `json:"format,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
	MaxLength  int                `json:"maxLength,omitempty"`
	Nullable   bool               `json:"nullable,omitempty"`
	Pattern    string             `json:"pattern,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Required   []string           `json:"required,omitempty"`
	Type       string             `json:"type,omitempty"`

	// Ref is populated if this JSON Schema is actually a JSON reference, and
	// it defines the location of the actual schema definition.
	Ref string `json:"$ref,omitempty"`

	XExpandableFields   *[]string           `json:"x-expandableFields,omitempty"`
	XExpansionResources *ExpansionResources `json:"x-expansionResources,omitempty"`
	XResourceID         string              `json:"x-resourceId,omitempty"`
}

func (s *Schema) String() string {
	js, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(js)
}

// UnmarshalJSON is a custom JSON unmarshaling implementation for Schema that
// provides better error messages instead of silently ignoring fields.
func (s *Schema) UnmarshalJSON(data []byte) error {
	var rawFields map[string]interface{}
	err := json.Unmarshal(data, &rawFields)
	if err != nil {
		return err
	}

	for _, supportedField := range supportedSchemaFields {
		delete(rawFields, supportedField)
	}
	for unsupportedField := range rawFields {
		return fmt.Errorf(
			"unsupported field in JSON schema: '%s'", unsupportedField)
	}

	// Define a second type that's identical to Schema, but distinct, so that when
	// we call json.Unmarshal it will call the default implementation of
	// unmarshalling a Schema object instead of recursively calling this
	// UnmarshalJSON function again.
	type schemaAlias Schema
	var inner schemaAlias
	err = json.Unmarshal(data, &inner)
	if err != nil {
		return err
	}
	*s = Schema(inner)

	return nil
}

// MediaType is a struct bucketing a request or response by media type in an
// OpenAPI specification.
type MediaType struct {
	Schema *Schema `json:"schema"`
}

// Operation is a struct representing a possible HTTP operation in an OpenAPI
// specification.
type Operation struct {
	Description string                  `json:"description"`
	OperationID string                  `json:"operation_id"`
	Parameters  []*Parameter            `json:"parameters"`
	RequestBody *RequestBody            `json:"requestBody"`
	Responses   map[StatusCode]Response `json:"responses"`
}

// Parameter is a struct representing a request parameter to an HTTP operation
// in an OpenAPI specification.
type Parameter struct {
	Description string  `json:"description"`
	In          string  `json:"in"`
	Name        string  `json:"name"`
	Required    bool    `json:"required"`
	Schema      *Schema `json:"schema"`
}

// Path is a type for an HTTP path in an OpenAPI specification.
type Path string

// RequestBody is a struct representing the body of a request in an OpenAPI
// specification.
type RequestBody struct {
	Content  map[string]MediaType `json:"content"`
	Required bool                 `json:"required"`
}

// Response is a struct representing the response of an HTTP operation in an
// OpenAPI specification.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content"`
}

// ResourceID is a type for the ID of a response resource in an OpenAPI
// specification.
type ResourceID string

// Spec is a struct representing an OpenAPI specification.
type Spec struct {
	Components Components                       `json:"components"`
	Info       *Info                            `json:"info"`
	Paths      map[Path]map[HTTPVerb]*Operation `json:"paths"`
}

//LoadSpec attempts to load the given embedded Spec.
func LoadSpec(specPath string) (*Spec, error) {
	var data []byte
	var err error

	if specPath == "" {
		// Load the spec information from go-bindata
		data, err = Asset("openapi/openapi/spec3.json")
	} else {
		if !isJSONFile(specPath) {
			return nil, fmt.Errorf("spec should come from a JSON file")
		}

		data, err = ioutil.ReadFile(specPath)
	}
	if err != nil {
		return nil, fmt.Errorf("error loading spec: %v", err)
	}

	var stripeSpec Spec
	err = json.Unmarshal(data, &stripeSpec)
	if err != nil {
		return nil, fmt.Errorf("error decoding spec: %v", err)
	}

	return &stripeSpec, nil
}

// StatusCode is a type for the response status code of an HTTP operation in an
// OpenAPI specification.
type StatusCode string
