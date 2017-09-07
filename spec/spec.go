package spec

import (
	"encoding/json"
)

type Components struct {
	Schemas map[string]*JSONSchema `json:"schemas"`
}

type ExpansionResources struct {
	OneOf []*JSONSchema `json:"oneOf" yaml:"oneOf"`
}

type Fixtures struct {
	Resources map[ResourceID]interface{} `json:"resources"`
}

type HTTPVerb string

type JSONSchema struct {
	AnyOf      []*JSONSchema          `json:"anyOf" yaml:"anyOf"`
	Enum       []string               `json:"enum" yaml:"enum"`
	Items      *JSONSchema            `json:"items" yaml:"items"`
	Nullable   bool                   `json:"nullable" yaml:"nullable"`
	Properties map[string]*JSONSchema `json:"properties" yaml:"properties"`
	Type       string                 `json:"type" yaml:"type"`

	// Ref is populated if this JSON Schema is actually a JSON reference, and
	// it defines the location of the actual schema definition.
	Ref string `json:"$ref" yaml:"$ref"`

	XExpandableFields   []string            `json:"x-expandableFields" yaml:"x-expandableFields"`
	XExpansionResources *ExpansionResources `json:"x-expansionResources" yaml:"x-expansionResources"`
	XResourceID         string              `json:"x-resourceId" yaml:"x-resourceId"`

	// RawFields stores a raw map of JSON schema fields to values. This is
	// useful when trying to interoperate with other libraries that use JSON
	// schemas.
	RawFields map[string]interface{}
}

func (s *JSONSchema) UnmarshalJSON(data []byte) error {
	type jsonSchema JSONSchema
	var inner jsonSchema
	err := json.Unmarshal(data, &inner)
	if err != nil {
		return err
	}
	*s = JSONSchema(inner)

	var rawFields map[string]interface{}
	err = json.Unmarshal(data, &rawFields)
	if err != nil {
		return err
	}
	s.RawFields = rawFields

	return nil
}

type MediaType struct {
	Schema *JSONSchema `json:"schema" yaml:"schema"`
}

type Operation struct {
	Description string                  `json:"description" yaml:"description"`
	OperationID string                  `json:"operation_id" yaml:"operation_id"`
	Parameters  []*Parameter            `json:"parameters" yaml:"parameters"`
	RequestBody *RequestBody            `json:"requestBody" yaml:"requestBody"`
	Responses   map[StatusCode]Response `json:"responses" yaml:"responses"`
}

type Parameter struct {
	Description string      `json:"description" yaml:"description"`
	In          string      `json:"in" yaml:"in"`
	Name        string      `json:"name" yaml:"name"`
	Required    bool        `json:"required" yaml:"required"`
	Schema      *JSONSchema `json:"schema" yaml:"schema"`
}

type Path string

type RequestBody struct {
	Content  map[string]MediaType `json:"content" yaml:"content"`
	Required bool                 `json:"required" yaml:"required"`
}

type Response struct {
	Description string                 `json:"description" yaml:"description"`
	Content     map[string]*JSONSchema `json:"content" yaml:"content"`
}

type ResourceID string

type Spec struct {
	Components  Components                       `json:"components" yaml:"components"`
	Paths       map[Path]map[HTTPVerb]*Operation `json:"paths" yaml:"paths"`
}

type StatusCode string
