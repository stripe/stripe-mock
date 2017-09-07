package spec

import (
  "github.com/lestrrat/go-jsschema"
  "github.com/lestrrat/go-jsval"
  "github.com/lestrrat/go-jsval/builder"
)

func GetValidatorForOpenAPI3Schema(oaiSchema *Schema) (*jsval.JSVal, error) {
  jsonSchemaAsJSON := getJSONSchemaForOpenAPI3Schema(oaiSchema)

  jsonSchema := schema.New()
  err := jsonSchema.Extract(jsonSchemaAsJSON)
  if err != nil {
    return nil, err
  }

  validatorBuilder := builder.New()
	validator, err := validatorBuilder.Build(jsonSchema)
	if err != nil {
		return nil, err
	}

  return validator, nil
}

// Given an OpenAPI 3 schema represented as JSON, returns an equivalent JSON
// Schema represented as JSON. The important difference between OpenAPI 3
// schemas and JSON schemas is that OpenAPI 3 uses "nullable: true" to mark
// values that can be null, whereas JSON schemas represent "null" as a type just
// like "string".
//
// This converter only handles the options that are supported by the spec.Schema
// type, and it must be updated when new options are supported.
func getJSONSchemaForOpenAPI3Schema(oai *Schema) (map[string]interface{}) {
  jss := make(map[string]interface{})
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
  if oai.Items != nil {
    jss["items"] = getJSONSchemaForOpenAPI3Schema(oai.Items)
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
