package spec

import (
	"encoding/json"
	"fmt"
)

//
// Public values
//

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

type Components struct {
	Schemas map[string]*Schema `json:"schemas" yaml:"schemas"`
}

type ExpansionResources struct {
	OneOf []*Schema `json:"oneOf" yaml:"oneOf"`
}

type Fixtures struct {
	Resources map[ResourceID]interface{} `json:"resources" yaml:"resources"`
}

type HTTPVerb string

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

	// This is currently a hint for the server-side so I haven't included it in
	// Schema yet. If we do start validating responses that come out of
	// stripe-mock, we may need to observe this as well.
	"x-stripeBypassValidation",
}

type Schema struct {
	// AdditionalProperties is either a `false` to indicate that no additional
	// properties in the object are allowed (beyond what's in Properties), or a
	// JSON schema that describes the expected format of any additional properties.
	//
	// We currently just read it as an `interface{}` because we're not using it
	// for anything right now.
	AdditionalProperties interface{} `json:"additionalProperties,omitempty" yaml:"additionalProperties"`

	AnyOf      []*Schema          `json:"anyOf,omitempty" yaml:"anyOf"`
	Enum       []interface{}      `json:"enum,omitempty" yaml:"enum"`
	Format     string             `json:"format,omitempty" yaml:"format"`
	Items      *Schema            `json:"items,omitempty" yaml:"items"`
	MaxLength  int                `json:"maxLength,omitempty" yaml:"maxLength"`
	Nullable   bool               `json:"nullable,omitempty" yaml:"nullable"`
	Pattern    string             `json:"pattern,omitempty" yaml:"pattern"`
	Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties"`
	Required   []string           `json:"required,omitempty" yaml:"required"`
	Type       string             `json:"type,omitempty" yaml:"type"`

	// Ref is populated if this JSON Schema is actually a JSON reference, and
	// it defines the location of the actual schema definition.
	Ref string `json:"$ref,omitempty" yaml:"$ref"`

	XExpandableFields   *[]string           `json:"x-expandableFields,omitempty" yaml:"x-expandableFields"`
	XExpansionResources *ExpansionResources `json:"x-expansionResources,omitempty" yaml:"x-expansionResources"`
	XResourceID         string              `json:"x-resourceId,omitempty" yaml:"x-resourceId"`
}

func (s *Schema) String() string {
	js, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(js)
}

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

type MediaType struct {
	Schema *Schema `json:"schema" yaml:"schema"`
}

type Operation struct {
	Description string                  `json:"description" yaml:"description"`
	OperationID string                  `json:"operation_id" yaml:"operation_id"`
	Parameters  []*Parameter            `json:"parameters" yaml:"parameters"`
	RequestBody *RequestBody            `json:"requestBody" yaml:"requestBody"`
	Responses   map[StatusCode]Response `json:"responses" yaml:"responses"`
}

type Parameter struct {
	Description string  `json:"description" yaml:"description"`
	In          string  `json:"in" yaml:"in"`
	Name        string  `json:"name" yaml:"name"`
	Required    bool    `json:"required" yaml:"required"`
	Schema      *Schema `json:"schema" yaml:"schema"`
}

type Path string

type RequestBody struct {
	Content  map[string]MediaType `json:"content" yaml:"content"`
	Required bool                 `json:"required" yaml:"required"`
}

type Response struct {
	Description string               `json:"description" yaml:"description"`
	Content     map[string]MediaType `json:"content" yaml:"content"`
}

type ResourceID string

type Spec struct {
	Components Components                       `json:"components" yaml:"components"`
	Paths      map[Path]map[HTTPVerb]*Operation `json:"paths" yaml:"paths"`
}

type StatusCode string
