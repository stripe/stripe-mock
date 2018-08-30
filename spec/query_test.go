package spec

import (
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestBuildQuerySchema(t *testing.T) {
	// Handles a normal case
	{
		operation := &Operation{
			Parameters: []*Parameter{
				{
					In:   ParameterQuery,
					Name: "name",
					Schema: &Schema{
						Type: TypeString,
					},
				},
			},
		}
		schema := BuildQuerySchema(operation)

		assert.Equal(t, false, schema.AdditionalProperties)
		assert.Equal(t, 1, len(schema.Properties))
		assert.Equal(t, 0, len(schema.Required))

		paramSchema := schema.Properties["name"]
		assert.Equal(t, TypeString, paramSchema.Type)
	}

	// A non-query parameter
	{
		operation := &Operation{
			Parameters: []*Parameter{
				{
					In:   ParameterPath,
					Name: "name",
				},
			},
		}
		schema := BuildQuerySchema(operation)

		assert.Equal(t, 0, len(schema.Properties))
	}

	// A required parameter
	{
		operation := &Operation{
			Parameters: []*Parameter{
				{
					In:       ParameterQuery,
					Name:     "name",
					Required: true,
					Schema: &Schema{
						Type: TypeString,
					},
				},
			},
		}
		schema := BuildQuerySchema(operation)

		assert.Equal(t, []string{"name"}, schema.Required)
	}

	// A query parameter with no schema
	{
		operation := &Operation{
			Parameters: []*Parameter{
				{
					In:   ParameterQuery,
					Name: "name",
				},
			},
		}
		schema := BuildQuerySchema(operation)

		paramSchema := schema.Properties["name"]
		assert.Equal(t, TypeObject, paramSchema.Type)
	}
}
