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
	Schemas map[string]*Schema `json:"schemas"`
}

type ExpansionResources struct {
	OneOf []*Schema `json:"oneOf"`
}

type Fixtures struct {
	Resources map[ResourceID]interface{} `json:"resources"`
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
	Schema *Schema `json:"schema"`
}

type Operation struct {
	Description string                  `json:"description"`
	OperationID string                  `json:"operation_id"`
	Parameters  []*Parameter            `json:"parameters"`
	RequestBody *RequestBody            `json:"requestBody"`
	Responses   map[StatusCode]Response `json:"responses"`
}

type Parameter struct {
	Description string  `json:"description"`
	In          string  `json:"in"`
	Name        string  `json:"name"`
	Required    bool    `json:"required"`
	Schema      *Schema `json:"schema"`
}

type Path string

type RequestBody struct {
	Content  map[string]MediaType `json:"content"`
	Required bool                 `json:"required"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content"`
}

type ResourceID string

type Spec struct {
	Components Components                       `json:"components"`
	Paths      map[Path]map[HTTPVerb]*Operation `json:"paths"`
}

type StatusCode string
