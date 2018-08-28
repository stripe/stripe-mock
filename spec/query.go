package spec

// BuildQuerySchema builds a JSON schema that will be used to validate query
// parameters on the incoming request. Unlike request bodies, OpenAPI puts
// query parameters in a different, non-JSON schema part of an operation.
func BuildQuerySchema(operation *Operation) *Schema {
	schema := &Schema{
		AdditionalProperties: false,
		Properties:           make(map[string]*Schema),
		Required:             make([]string, 0),
		Type:                 TypeObject,
	}

	if operation.Parameters == nil {
		return schema
	}

	for _, param := range operation.Parameters {
		if param.In != ParameterQuery {
			continue
		}

		paramSchema := param.Schema
		if paramSchema == nil {
			paramSchema = &Schema{Type: TypeObject}
		}
		schema.Properties[param.Name] = paramSchema

		if param.Required {
			schema.Required = append(schema.Required, param.Name)
		}
	}

	return schema
}
