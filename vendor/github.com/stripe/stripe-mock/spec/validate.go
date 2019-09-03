package spec

import (
	"github.com/lestrrat-go/jsschema"
	"github.com/lestrrat-go/jsval"
	"github.com/lestrrat-go/jsval/builder"
)

// ComponentsForValidation is a collection of components for an OpenAPI
// specification that's been translated into equivalent JSON Schemas.
type ComponentsForValidation struct {
	root interface{}
}

// GetValidatorForOpenAPI3Schema gets a JSON Schema validator for a given
// OpenAPI specification and set of JSON Schema components.
func GetValidatorForOpenAPI3Schema(oaiSchema *Schema, components *ComponentsForValidation) (*jsval.JSVal, error) {
	jsonSchemaAsJSON := getJSONSchemaForOpenAPI3Schema(oaiSchema)

	jsonSchema := schema.New()
	err := jsonSchema.Extract(jsonSchemaAsJSON)
	if err != nil {
		return nil, err
	}

	if components == nil {
		components = &ComponentsForValidation{root: make(map[string]interface{})}
	}

	validatorBuilder := builder.New()
	validator, err := validatorBuilder.BuildWithCtx(jsonSchema, components.root)
	if err != nil {
		return nil, err
	}

	return validator, nil
}

// GetComponentsForValidation translates a collection of components for an
// OpenAPI specification into equivalent JSON schemas.
//
// See also the comment on getJSONSchemaForOpenAPI3Schema.
func GetComponentsForValidation(components *Components) *ComponentsForValidation {
	jsonSchemas := make(map[string]interface{})
	for name, oaiSchema := range components.Schemas {
		jsonSchemas[name] = getJSONSchemaForOpenAPI3Schema(oaiSchema)
	}
	return &ComponentsForValidation{
		root: map[string]interface{}{
			"components": map[string]interface{}{
				"schemas": jsonSchemas,
			},
		},
	}
}

// Given an OpenAPI 3 schema represented as JSON, returns an equivalent JSON
// Schema represented as JSON. The important difference between OpenAPI 3
// schemas and JSON schemas is that OpenAPI 3 uses "nullable: true" to mark
// values that can be null, whereas JSON schemas represent "null" as a type just
// like "string".
//
// This converter only handles the options that are supported by the spec.Schema
// type, and it must be updated when new options are supported.
func getJSONSchemaForOpenAPI3Schema(oai *Schema) map[string]interface{} {
	jss := make(map[string]interface{})
	if oai.AdditionalProperties != nil {
		// We currently don't decode `AdditionalProperties` into a custom
		// struct, so it's a pretty direct JSON representation. Just set it
		// directly.
		jss["additionalProperties"] = oai.AdditionalProperties
	}
	if len(oai.AnyOf) != 0 {
		var jssAnyOf = make([]interface{}, len(oai.AnyOf))
		for index, oaiSubschema := range oai.AnyOf {
			jssAnyOf[index] = getJSONSchemaForOpenAPI3Schema(oaiSubschema)
		}
		if oai.Nullable {
			jssAnyOf = append(jssAnyOf, map[string]interface{}{"const": nil})
		}
		jss["anyOf"] = jssAnyOf
	}
	if len(oai.Enum) != 0 {
		var jssEnum = make([]interface{}, len(oai.Enum))
		for index, oaiValue := range oai.Enum {
			jssEnum[index] = oaiValue
		}
		jss["enum"] = jssEnum
	}
	if oai.Format != "" {
		// Note that the major format that will be seen here, unix-time, will
		// not be supported by the validator we're using -- we should probably
		// see if we can support that properly.
		jss["format"] = oai.Format
	}
	if oai.Items != nil {
		jss["items"] = getJSONSchemaForOpenAPI3Schema(oai.Items)
	}
	if oai.MaxLength != 0 {
		jss["maxLength"] = oai.MaxLength
	}
	if oai.Pattern != "" {
		jss["pattern"] = oai.Pattern
	}
	if len(oai.Properties) != 0 {
		var jssProperties = make(map[string]interface{})
		for key, oaiSubschema := range oai.Properties {
			jssProperties[key] = getJSONSchemaForOpenAPI3Schema(oaiSubschema)
		}
		jss["properties"] = jssProperties
	}
	if len(oai.Required) != 0 {
		var jssRequired = make([]interface{}, len(oai.Required))
		for index, oaiValue := range oai.Required {
			jssRequired[index] = oaiValue
		}
		jss["required"] = jssRequired
	}
	if oai.Type != "" {
		if oai.Nullable {
			jss["type"] = []interface{}{oai.Type, "null"}
		} else {
			jss["type"] = oai.Type
		}
	}
	if oai.Ref != "" {
		jss["$ref"] = oai.Ref
	}
	return jss
}
