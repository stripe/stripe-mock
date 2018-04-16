package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/stripe/stripe-mock/spec"
)

// GenerateParams is a parameters structure that's used to invoke Generate and
// its associated methods.
//
// This structure exists to avoid runaway paramete inflation for the Generate
// function, so that we can document individual parameters in a more organized
// way, and because it can conveniently encapsulate some unexported fields that
// Generate uses to track its progress.
type GenerateParams struct {
	// Expansions are the requested expansions for the current level of generation.
	//
	// nil if no expansions were requested, or we've recursed to a level where
	// none of the original expansions applied.
	Expansions *ExpansionLevel

	// ID is the primary identifier for the object being generated. If one was
	// provided, it's used to replace some information from our fixtures.
	//
	// nil if there is no replacement for the ID.
	//
	// The value of this field is expected to stay stable across all levels of
	// recursion.
	ID *string

	// RequestPath is the path of the URL being requested which we're
	// generating data for. It's used to populate the url property of any
	// nested lists that we generate.
	//
	// The value of this field is expected to stay stable across all levels of
	// recursion.
	RequestPath string

	//
	// Private fields
	//

	// Schema representing the object that we're trying to generate.
	//
	// The value of this field will change as Generate recurses to the target
	// schema at that level of recursion.
	//
	// This field is required.
	Schema *spec.Schema

	// context is a breadcrumb trail that's added to as Generate recurses. It's
	// not important for the final result, but is very useful for debugging.
	context string

	// doReplaceID indicates that we should try to replace an ID at this level
	// of recursion. It's useful because since we're replacing only an object's
	// primary ID, we only want to do so at the top level of a generated object
	// (this is falsed for most levels of recursion, except in the cases where
	// recursion is used to follow something like an anyOf branch to generate
	// data for the top-level object).
	doReplaceID bool

	// example is a valid data sample for the target schema at this level of
	// recursion.
	//
	// nil means that were was no sample available. A valueWrapper instance
	// with an embedded nil means that there is a sample, and it's nil/null.
	example *valueWrapper

	// replacedID is the old value of an ID field that's had its value replaced
	// by ID. This is used so that we can look for other instances of this
	// replaced ID, and also replace them (useful in case the same ID was
	// reference in a list URL or in an embedded subresource).
	//
	// nil if no ID has been replaced.
	replacedID *string
}

// DataGenerator generates fixture response data based off a response schema, a
// set of definitions, and a fixture store.
type DataGenerator struct {
	definitions map[string]*spec.Schema
	fixtures    *spec.Fixtures
}

// Generate generates a fixture response.
func (g *DataGenerator) Generate(params *GenerateParams) (interface{}, error) {
	// This just makes our context message readable in case there was no
	// request path specified.
	requestPathDisplay := params.RequestPath
	if requestPathDisplay == "" {
		requestPathDisplay = "(empty request path)"
	}

	return g.generateInternal(&GenerateParams{
		Expansions:  params.Expansions,
		ID:          params.ID,
		RequestPath: params.RequestPath,
		Schema:      params.Schema,

		context:     fmt.Sprintf("Responding to %s:\n", requestPathDisplay),
		doReplaceID: true,
		example:     nil,
		replacedID:  nil,
	})
}

// generateInternal encompasses all the generation logic. It's separate from
// Generate only so that Generate can seed it with a little bit of information.
func (g *DataGenerator) generateInternal(params *GenerateParams) (interface{}, error) {
	// This is a bit of a mess. We don't have an elegant fully-general approach to
	// generating examples, just a bunch of specific cases that we know how to
	// handle. If we find ourselves in a situation that doesn't match any of the
	// cases, then we fall through to the end of the function and panic().
	// Obviously this is fragile, so we have a unit test that makes sure it works
	// correctly on every resource; hopefully this will at least allow us to catch
	// any errors in advance.

	schema, context, err := g.maybeDereference(params.Schema, params.context)
	if err != nil {
		return nil, err
	}

	// Determine if the requested expansions are possible
	if params.Expansions != nil && schema.XExpandableFields != nil {
		for key := range params.Expansions.expansions {
			if sort.SearchStrings(*schema.XExpandableFields, key) ==
				len(*schema.XExpandableFields) {
				return nil, errExpansionNotSupported
			}
		}
	}

	example := params.example
	if (example == nil || example.value == nil) && schema.XResourceID != "" {
		// Use the fixture as our example. (Note that if the caller gave us a
		// non-trivial example, we prefer it instead, because it's probably more
		// relevant in context.)
		fixture, ok := g.fixtures.Resources[spec.ResourceID(schema.XResourceID)]
		if !ok {
			panic(fmt.Sprintf("%sMissing fixture for: %s", context, schema.XResourceID))
		}

		example = &valueWrapper{value: fixture}
		context = fmt.Sprintf("%sUsing fixture '%s':\n", context, schema.XResourceID)
	}

	if schema.XExpansionResources != nil {
		if params.Expansions != nil {
			// We're expanding this specific object
			return g.generateInternal(&GenerateParams{
				Expansions:  params.Expansions,
				ID:          params.ID,
				RequestPath: params.RequestPath,
				Schema:      schema.XExpansionResources.OneOf[0],

				context:     fmt.Sprintf("%sExpanding optional expandable field:\n", context),
				doReplaceID: false,
				example:     nil,
				replacedID:  params.replacedID,
			})
		} else {
			// We're not expanding this specific object. Our example should be of the
			// unexpanded form, which is the first branch of the AnyOf
			return g.generateInternal(&GenerateParams{
				Expansions:  params.Expansions,
				ID:          params.ID,
				RequestPath: params.RequestPath,
				Schema:      schema.AnyOf[0],

				context:     fmt.Sprintf("%sNot expanding optional expandable field:\n", context),
				doReplaceID: false,
				example:     example,
				replacedID:  params.replacedID,
			})
		}
	}

	if len(schema.AnyOf) == 1 && schema.Nullable {
		if example != nil && example.value == nil {
			if params.Expansions == nil {
				return nil, nil
			}
		} else {
			// Since there's only one subschema, we can confidently recurse into it
			return g.generateInternal(&GenerateParams{
				Expansions:  params.Expansions,
				ID:          params.ID,
				RequestPath: params.RequestPath,
				Schema:      schema.AnyOf[0],

				context:     fmt.Sprintf("%sChoosing only branch of anyOf:\n", context),
				doReplaceID: params.doReplaceID,
				example:     example,
				replacedID:  params.replacedID,
			})
		}
	}

	if len(schema.AnyOf) != 0 {
		// Just generate an example of the first subschema. Note that we don't pass
		// in any example, even if we have an example available, because we don't
		// know which branch of the AnyOf the example corresponds to.
		return g.generateInternal(&GenerateParams{
			Expansions:  params.Expansions,
			ID:          params.ID,
			RequestPath: params.RequestPath,
			Schema:      schema.AnyOf[0],

			context:     fmt.Sprintf("%sChoosing first branch of anyOf:\n", context),
			doReplaceID: params.doReplaceID,
			example:     nil,
			replacedID:  params.replacedID,
		})
	}

	if isListResource(schema) {
		// We special-case list resources and always fill in the list with at least
		// one item of data, regardless of what was present in the example
		listData, err := g.generateListResource(&GenerateParams{
			Expansions:  params.Expansions,
			ID:          params.ID,
			RequestPath: params.RequestPath,
			Schema:      schema,

			context:     context,
			doReplaceID: params.doReplaceID,
			example:     example,
			replacedID:  params.replacedID,
		})
		return listData, err
	}

	// Generate a synthethic schema as a last ditch effort
	if example == nil && schema.XResourceID == "" {
		example = &valueWrapper{value: generateSyntheticFixture(schema, context)}

		context = fmt.Sprintf("%sGenerated synthetic fixture: %+v\n", context, schema)

		if verbose {
			// We list properties here because the schema might not have a
			// better name to identify it with.
			fmt.Printf("Generated synthetic fixture with properties: %s\n",
				stringOrEmpty(propertyNames(schema)))
		}
	}

	if example == nil {
		// If none of the above conditions met, we've run out of ways of generating
		// examples from scratch, so we can only raise an error.
		panic(fmt.Sprintf("%sCannot find or generate example for: %s", context, schema))
	}

	if example.value == nil {
		if params.Expansions != nil {
			panic(fmt.Sprintf("%sWe were asked to expand a key, but our example "+
				"has null for that key.", context))
		}
		return nil, nil
	}

	// If we replaced a primary object ID, then also replace any of values that
	// we happen to find which had the same value as it. These will usually be
	// IDs in child objects that reference the parent.
	//
	// For example, a charge may contain a sublist of refunds. If we replaced
	// the charge's ID, we also want to replace that charge ID in every one of
	// the child refunds.
	if params.replacedID != nil && schema.Type == "string" {
		valStr, ok := example.value.(string)
		if ok && valStr == *params.replacedID {
			example = &valueWrapper{value: *params.ID}
		}
	}

	if schema.Type == "boolean" || schema.Type == "integer" ||
		schema.Type == "number" || schema.Type == "string" {
		return example.value, nil
	}

	if schema.Type == "object" && schema.Properties == nil {
		// For a generic object type with no particular properties specified, we
		// assume it must not contain any expandable fields or list resources
		return example.value, nil
	}

	if schema.Type == "array" {
		// For lists that aren't contained in a list-object, we assume they do not
		// contain any expandable fields or list resources
		return example.value, nil
	}

	if schema.Type == "object" && schema.Properties != nil {
		exampleMap, ok := example.value.(map[string]interface{})
		if !ok {
			panic(fmt.Sprintf(
				"%sSchema is an object:\n%s\n\nBut example is (type: %v):\n%s",
				context, schema, reflect.TypeOf(example.value), example.value))
		}

		resultMap := make(map[string]interface{})

		// We might have obtained an ID for the object from an extracted path
		// parameter. If we did, fill it in. Note that this only occurs at the
		// top level of recursion because any ID fields we find at other levels
		// are likely for other objects.
		//
		// If we do replace an ID, extract the old one so that we can inject it
		// into list URLs from our fixtures.
		//
		// This replacement must occur before iterating through the loop below
		// because we might also use the new ID to replace other values in the
		// object as well.
		replacedID := params.replacedID
		if params.doReplaceID && params.ID != nil {
			_, ok := schema.Properties["id"]
			if ok {
				idValue, ok := exampleMap["id"]
				if ok {
					idValueStr := idValue.(string)
					replacedID = &idValueStr
					resultMap["id"] = *params.ID

					if verbose {
						fmt.Printf("Found ID to replace; previous: '%s' new: '%s'\n",
							*replacedID, *params.ID)
					}
				}
			}
		}

		for key, subSchema := range schema.Properties {
			// If these conditions are met this was handled above. Skip it.
			if params.doReplaceID && key == "id" && replacedID != nil {
				continue
			}

			var subExpansions *ExpansionLevel
			if params.Expansions != nil {
				subExpansions = params.Expansions.expansions[key]
				if subExpansions == nil && params.Expansions.wildcard {
					// No expansion was provided for this key but the wildcard bit is set,
					// so make a fake expansion
					subExpansions = &ExpansionLevel{
						expansions: make(map[string]*ExpansionLevel),
						wildcard:   false,
					}
				}
			}

			var subvalueWrapper *valueWrapper
			subvalueWrapperValue, exampleHasKey := exampleMap[key]
			if exampleHasKey {
				subvalueWrapper = &valueWrapper{value: subvalueWrapperValue}
			}

			if !exampleHasKey && subExpansions == nil {
				// If the example omitted this key, then so do we; unless we were asked
				// to expand the key, in which case we'll have to generate an example
				// from scratch.
				continue
			}

			subValue, err := g.generateInternal(&GenerateParams{
				Expansions:  subExpansions,
				ID:          params.ID,
				RequestPath: params.RequestPath,
				Schema:      subSchema,

				context:     fmt.Sprintf("%sIn property '%s' of object:\n", context, key),
				doReplaceID: false,
				example:     subvalueWrapper,
				replacedID:  replacedID,
			})
			if err != nil {
				return nil, err
			}
			resultMap[key] = subValue
		}

		return resultMap, nil
	}

	// If the schema is of the format we expect, this shouldn't ever happen.
	panic(fmt.Sprintf(
		"%sEncountered unusual scenario:\nschema=%s\nexample=%+v",
		context, schema, example))
}

func (g *DataGenerator) maybeDereference(schema *spec.Schema, context string) (*spec.Schema, string, error) {
	if schema.Ref != "" {
		definition := definitionFromJSONPointer(schema.Ref)

		newSchema, ok := g.definitions[definition]
		if !ok {
			panic(fmt.Sprintf("Couldn't dereference: %v", schema.Ref))
		}
		context = fmt.Sprintf("%sDereferencing '%s':\n", context, schema.Ref)
		schema = newSchema
	}
	return schema, context, nil
}

func (g *DataGenerator) generateListResource(params *GenerateParams) (interface{}, error) {
	var itemExpansions *ExpansionLevel
	if params.Expansions != nil {
		itemExpansions = params.Expansions.expansions["data"]
	}

	itemData, err := g.generateInternal(&GenerateParams{
		Expansions:  itemExpansions,
		ID:          params.ID,
		RequestPath: params.RequestPath,
		Schema:      params.Schema.Properties["data"].Items,

		context:     fmt.Sprintf("%sPopulating list resource:\n", params.context),
		doReplaceID: false,
		example:     nil,
		replacedID:  params.replacedID,
	})
	if err != nil {
		return nil, err
	}

	// This is written to hopefully be a little more forward compatible in that
	// it respects the list properties dictated by the included schema rather
	// than assuming its own.
	listData := make(map[string]interface{})
	for key, subSchema := range params.Schema.Properties {
		var val interface{}
		switch key {
		case "data":
			val = []interface{}{itemData}
		case "has_more":
			val = false
		case "object":
			val = "list"
		case "total_count":
			val = 1
		case "url":
			var url string
			if strings.HasPrefix(subSchema.Pattern, "^") {
				// Many list resources have a URL pattern of the form "^/v1/whatevers";
				// we cut off the "^" to leave the URL.
				url = subSchema.Pattern[1:]
			} else if params.example != nil {
				// If an example was provided, we can assume it has the correct format
				example := params.example.value.(map[string]interface{})
				url = example["url"].(string)
			} else {
				url = params.RequestPath
			}

			// Potentially replace a primary ID in the URL of a list so that
			// requests against it may be consistent. For example, if
			// `/v1/charges/ch_123` was requested, we'd want the refunds list
			// within the returned object to have a URL like
			// `/v1/charges/ch_123/refunds`.
			if params.replacedID != nil {
				val = strings.Replace(url, *params.replacedID, *params.ID, 1)
			} else {
				val = url
			}
		default:
			val = nil
		}
		listData[key] = val
	}
	return listData, nil
}

//
// Private values
//

var errExpansionNotSupported = fmt.Errorf("Expansion not supported")

//
// Private types
//

// valueWrapper wraps an example value that we're generating.
//
// It exists so that we can make a distinction between an example that we don't
// have (where `valueWrapper` itself is `nil`) from one where we have an
// example, but it has a `null` value (where we'd have `valueWrapper{value:
// nil}`).
type valueWrapper struct {
	value interface{}
}

//
// Private functions
//

// definitionFromJSONPointer extracts the name of a JSON schema definition from
// a JSON pointer, so "#/components/schemas/charge" would become just "charge".
// This is a simplified workaround to avoid bringing in JSON schema
// infrastructure because we can guarantee that the spec we're producing will
// take a certain shape. If this gets too hacky, it will be better to put a more
// legitimate JSON schema parser in place.
func definitionFromJSONPointer(pointer string) string {
	parts := strings.Split(pointer, "/")

	if len(parts) != 4 ||
		parts[0] != "#" ||
		parts[1] != "components" ||
		parts[2] != "schemas" {
		panic(fmt.Sprintf("Expected '#/components/schemas/...' but got '%v'", pointer))
	}
	return parts[3]
}

// generateSyntheticFixture generates a synthetic fixture for the given schema
// by examining its properties and returning default values for each.
//
// This is useful in cases where we don't have a valid fixture for some object.
// That could happen for a prerelease object or in cases where an expansion has
// been requested for an embedded object that doesn't occur at the top level of
// the API.
//
// This function calls itself recursively by initially iterating through every
// property in an object schema, then recursing and returning values for
// embedded objects and scalars.
func generateSyntheticFixture(schema *spec.Schema, context string) interface{} {
	context = fmt.Sprintf("%sGenerating synthetic fixture: %+v\n", context, schema)

	// Return the minimum viable object by returning nil/null for a nullable
	// property.
	if schema.Nullable {
		return nil
	}

	// Return a member of an enum if one is available because it's probably
	// going to be a more realistic value.
	if len(schema.Enum) > 0 {
		return schema.Enum[0]
	}

	if len(schema.AnyOf) > 0 {
		for _, subSchema := range schema.AnyOf {
			// We don't handle dereferencing here right now, but it's plausible
			if subSchema.Ref != "" {
				continue
			}
			return generateSyntheticFixture(subSchema, context)
		}
		panic(fmt.Sprintf("%sCouldn't find an anyOf branch to take", context))
	}

	switch schema.Type {
	case spec.TypeArray:
		return []string{}

	case spec.TypeBoolean:
		return true

	case spec.TypeInteger:
		return 0

	case spec.TypeNumber:
		return 0.0

	case spec.TypeObject:
		fixture := make(map[string]interface{})
		for property, subSchema := range schema.Properties {
			// Return the minimum viable object by not including properties
			// that are not necessary for a valid object.
			if !isRequiredProperty(schema, property) {
				continue
			}

			fixture[property] = generateSyntheticFixture(subSchema, context)
		}
		return fixture

	case spec.TypeString:
		return ""
	}

	panic(fmt.Sprintf("%sUnhandled type: %s", context, stringOrEmpty(schema.Type)))
}

func isListResource(schema *spec.Schema) bool {
	if schema.Type != "object" || schema.Properties == nil {
		return false
	}

	object, ok := schema.Properties["object"]
	if !ok || object.Enum == nil || object.Enum[0] != "list" {
		return false
	}

	data, ok := schema.Properties["data"]
	if !ok || data.Items == nil {
		return false
	}

	return true
}

// isRequiredProperty checks whether the given property name is required for
// the given schema. Note that this assumes that the schema is of type object
// because that would be semantic nonsense for any other type.
func isRequiredProperty(schema *spec.Schema, name string) bool {
	for _, property := range schema.Required {
		if name == property {
			return true
		}
	}
	return false
}

// propertyNames returns the names of all properties of a schema joined
// together and comma-separated.
//
// This is useful for printing debugging information.
func propertyNames(schema *spec.Schema) string {
	var names []string
	for name := range schema.Properties {
		names = append(names, name)
	}

	// Sort just so we can have stable output to test against (the order at
	// which keys will be iterated in the map is undefined).
	sort.Strings(names)

	return strings.Join(names, ", ")
}

// stringOrEmpty returns the string given as parameter, or the string "(empty)"
// if the string was empty.
//
// This is useful in cases like logging to make sure that something is always
// printed on screen (instead of a strangely truncated sentence for an empty
// value).
func stringOrEmpty(s string) string {
	if s == "" {
		return "(empty)"
	}
	return s
}
