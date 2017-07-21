package spec

import (
	"encoding/json"
)

type Fixtures struct {
	Resources map[ResourceID]interface{} `json:"resources"`
}

type HTTPVerb string

type JSONSchema struct {
	Enum       []string               `json:"enum"`
	Items      *JSONSchema            `json:"items"`
	OneOf      []*JSONSchema          `json:"oneOf"`
	Properties map[string]*JSONSchema `json:"properties"`
	Type       []string               `json:"type"`

	// Ref is populated if this JSON Schema is actually a JSON reference, and
	// it defines the location of the actual schema definition.
	Ref string `json:"$ref"`

	XExpandableFields   []string    `json:"x-expandableFields"`
	XExpansionResources *JSONSchema `json:"x-expansionResources"`
	XResourceID         string      `json:"x-resourceId"`

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

type Parameter struct {
	Description string      `json:"description"`
	In          string      `json:"in"`
	Name        string      `json:"name"`
	Required    bool        `json:"required"`
	Schema      *JSONSchema `json:"schema"`
}

type Method struct {
	Description string                  `json:"description"`
	OperationID string                  `json:"operation_id"`
	Parameters  []*Parameter            `json:"parameters"`
	Responses   map[StatusCode]Response `json:"responses"`
}

type Path string

type Response struct {
	Description string      `json:"description"`
	Schema      *JSONSchema `json:"schema"`
}

type ResourceID string

type Spec struct {
	Definitions map[string]*JSONSchema        `json:"definitions"`
	Paths       map[Path]map[HTTPVerb]*Method `json:"paths"`
}

type StatusCode string
