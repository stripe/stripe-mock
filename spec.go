package main

type Fixtures struct {
	Resources map[ResourceID]interface{} `json:"resources"`
}

type HTTPVerb string

type JSONSchema struct {
	Enum       []string               `json:"enum"`
	Items      *JSONSchema            `json:"items"`
	Properties map[string]*JSONSchema `json:"properties"`
	Type       []string               `json:"type"`

	// Ref is populated if this JSON Schema is actually a JSON reference, and
	// it defines the location of the actual schema definition.
	Ref string `json:"$ref"`

	XResourceID string `json:"x-resourceId"`
}

type OpenAPIParameter struct {
	Description string      `json:"description"`
	In          string      `json:"in"`
	Name        string      `json:"name"`
	Required    bool        `json:"required"`
	Schema      *JSONSchema `json:"schema"`
}

type OpenAPIMethod struct {
	Description string                                `json:"description"`
	OperationID string                                `json:"operation_id"`
	Parameters  []OpenAPIParameter                    `json:"parameters"`
	Responses   map[OpenAPIStatusCode]OpenAPIResponse `json:"responses"`
}

type OpenAPIPath string

type OpenAPIResponse struct {
	Description string      `json:"description"`
	Schema      *JSONSchema `json:"schema"`
}

type OpenAPISpec struct {
	Definitions map[string]*JSONSchema                      `json:"definitions"`
	Paths       map[OpenAPIPath]map[HTTPVerb]*OpenAPIMethod `json:"paths"`
}

type OpenAPIStatusCode string

type ResourceID string
