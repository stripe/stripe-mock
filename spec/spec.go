package spec

import (
	"encoding/json"
	"fmt"
)

type Components struct {
	Schemas map[string]*Schema `json:"schemas"`
}

type ExpansionResources struct {
	OneOf []*Schema `json:"oneOf" yaml:"oneOf"`
}

type Fixtures struct {
	Resources map[ResourceID]interface{} `json:"resources"`
}

type HTTPVerb string

type Schema struct {
	AnyOf      []*Schema          `json:"anyOf,omitempty" yaml:"anyOf"`
	Enum       []string           `json:"enum,omitempty" yaml:"enum"`
	Items      *Schema            `json:"items,omitempty" yaml:"items"`
	Nullable   bool               `json:"nullable,omitempty" yaml:"nullable"`
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

	// If you update this, be sure to also update getJSONSchemaForOpenAPI3Schema
	// to pass the fields through to the JSON validator
	supportedFields := []string{
		"anyOf", "enum", "items", "nullable", "properties", "required", "type",
		"x-expandableFields", "x-expansionResources", "x-resourceId", "$ref",
		"title", "description",
	}
	for _, supportedField := range supportedFields {
		delete(rawFields, supportedField)
	}
	for unsupportedField := range rawFields {
		return fmt.Errorf(
			"unsupported field in JSON schema: '%s'", unsupportedField)
	}

	type schema2 Schema
	var inner schema2
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
