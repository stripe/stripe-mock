package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompilePath(t *testing.T) {
	assert.Equal(t, `/v1/charges`,
		compilePath(OpenAPIPath("/v1/charges")).String())
	assert.Equal(t, `/v1/charges/(?P<id>\w+)`,
		compilePath(OpenAPIPath("/v1/charges/{id}")).String())
}

func TestDefinitionFromJSONPointer(t *testing.T) {
	definition, err := definitionFromJSONPointer("#/definitions/charge")
	assert.Nil(t, err)
	assert.Equal(t, "charge", definition)
}
