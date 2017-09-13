package spec

import (
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestValidator_Simple(t *testing.T) {
	schema := Schema{
		Type: "string",
	}
	v, err := GetValidatorForOpenAPI3Schema(&schema, nil)
	assert.NoError(t, err)
	assert.NoError(t, v.Validate("hello"))
	assert.Error(t, v.Validate(nil))
	assert.Error(t, v.Validate(123))
}

func TestValidator_Nullable(t *testing.T) {
	schema := Schema{
		Type:     "string",
		Nullable: true,
	}
	v, err := GetValidatorForOpenAPI3Schema(&schema, nil)
	assert.NoError(t, err)
	assert.NoError(t, v.Validate("hello"))
	assert.NoError(t, v.Validate(nil))
	assert.Error(t, v.Validate(123))
}

func TestValidator_Reference(t *testing.T) {
	fooSchema := Schema{
		Type: "string",
	}
	components := Components{
		Schemas: map[string]*Schema{
			"foo": &fooSchema,
		},
	}
	componentsForValidation := GetComponentsForValidation(&components)
	fooRefSchema := Schema{
		Ref: "#/components/schemas/foo",
	}
	v, err := GetValidatorForOpenAPI3Schema(&fooRefSchema, componentsForValidation)
	assert.NoError(t, err)
	assert.NoError(t, v.Validate("hello"))
	assert.Error(t, v.Validate(123))
}
