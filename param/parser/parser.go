package parser

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseFormString takes a form-encoded string and it turns it into a usable
// set of parameters that an application can consume. These parameters use
// "Rack-style" conventions for encoding complex types like arrays and maps,
// which is how the Stripe API decodes data. These complex types are the more
// difficult part of the problem and what makes this package fairly complicated
// to implemented.
func ParseFormString(s string) (map[string]interface{}, error) {
	values, err := parseForm(s)
	if err != nil {
		return nil, err
	}
	return parseFormValues(values)
}

//
// ---
//

const (
	keyTypeArray = iota
	keyTypeMap
	keyTypeRaw
)

// keyPart is an interface for a struct that represents a "part" of a parameter
// key. For example, we might say that in `obj[]` the `[]` is an array part.
type keyPart interface {
	KeyType() int
	Content() string
}

// keyType represents an array part of a parameter key, like the `[]` in
// `obj[][foo]`.
type keyArray struct {
}

func (k *keyArray) KeyType() int {
	return keyTypeArray
}

func (k *keyArray) Content() string {
	panic("keyArray doesn't support Content")
}

// keyMap represents a map part of a parameter key, like the `[foo]` in
// `obj[][foo]`.
type keyMap struct {
	content string
}

func (k *keyMap) KeyType() int {
	return keyTypeMap
}

func (k *keyMap) Content() string {
	return k.content
}

// keyRaw represents the raw name of a parameter key, like the `obj` in
// `obj[][foo]`.
type keyRaw struct {
	content string
}

func (k *keyRaw) KeyType() int {
	return keyTypeRaw
}

func (k *keyRaw) Content() string {
	return k.content
}

func parseKey(key string) []keyPart {
	var keyParts []keyPart
	var c rune
	var i int
	var mapContent []rune
	var rawContent []rune

	keyRunes := []rune(key)

raw:
	if i >= len(keyRunes) {
		goto finished
	}

	c = keyRunes[i]
	i++

	if c == '[' {
		if len(rawContent) > 0 {
			keyParts = append(keyParts, &keyRaw{content: string(rawContent)})
			rawContent = nil
		}

		goto inMapOrArray
	}

	rawContent = append(rawContent, c)
	goto raw

inMapOrArray:
	if i >= len(keyRunes) {
		goto finished
	}

	c = keyRunes[i]
	i++

	if c == ']' {
		keyParts = append(keyParts, &keyArray{})
		goto raw
	}

	mapContent = append(mapContent, c)

	// Fall through to inMap

inMap:
	if i >= len(keyRunes) {
		goto finished
	}

	c = keyRunes[i]
	i++

	if c == ']' {
		keyParts = append(keyParts, &keyMap{content: string(mapContent)})
		mapContent = nil
		goto raw
	}

	mapContent = append(mapContent, c)
	goto inMap

finished:
	if len(keyParts) == 0 && len(rawContent) > 0 {
		keyParts = append(keyParts, &keyRaw{content: string(rawContent)})
		rawContent = nil
	}

	return keyParts
}

func buildParamStructure(key string, parts []keyPart, value string) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	if len(parts) == 0 {
		params[key] = value
		return params, nil
	}

	part := parts[0]

	switch part.KeyType() {
	case keyTypeArray:
		subParams, err := buildParamStructure("dummy", parts[1:], value)
		if err != nil {
			return nil, err
		}
		params[key] = []interface{}{subParams["dummy"]}

	case keyTypeMap:
		subParams, err := buildParamStructure(part.Content(), parts[1:], value)
		if err != nil {
			return nil, err
		}
		params[key] = subParams

	default:
		return nil, fmt.Errorf("Invalid key. Raw content can't be mixed in after arrays and maps.")
	}

	return params, nil
}

func maybeCollapseArrays(arr1, arr2 []interface{}) bool {
	//fmt.Printf("maybe collapse arrays %+v %+v\n", arr1, arr2)

	if len(arr2) != 1 {
		panic("Expected array with length exactly 1")
	}

	// arr1's merge candidate is its last element only
	arr1Map, ok := arr1[len(arr1)-1].(map[string]interface{})
	if !ok {
		return false
	}

	arr2Map, ok := arr2[0].(map[string]interface{})
	if !ok {
		return false
	}

	// If any of the keys in arr2's map are already in arr1's map, then don't
	// merge we consider this a new map.
	for key := range arr2Map {
		val, ok := arr1Map[key]
		if ok {
			_, okArr := val.([]interface{})
			_, okMap := val.(map[string]interface{})
			if !okArr && !okMap {
				return false
			}
		}
	}

	mergeMapsRecursive(arr1Map, arr2Map)
	return true
}

func mergeMapsRecursive(map1, map2 map[string]interface{}) {
	for key, val2 := range map2 {
		val1, ok := map1[key]
		if ok {
			val1Map, ok := val1.(map[string]interface{})
			if ok {
				val2Map, ok := val2.(map[string]interface{})
				if ok {
					mergeMapsRecursive(val1Map, val2Map)
					continue
				}
			}

			val1Arr, ok := val1.([]interface{})
			if ok {
				val2Arr, ok := val2.([]interface{})
				if ok {
					ok := maybeCollapseArrays(val1Arr, val2Arr)
					if !ok {
						map1[key] = append(val1Arr, val2Arr...)
					}
					continue
				}
			}
		}

		// If not an array or map, or we couldn't reconcile types between the
		// two maps, simply set the key in map1 to the value from map2.
		map1[key] = val2
	}
}

// formPair is a key/value pair as extracted from a form-encoded string. For
// example, "a=b" is the pair [a, b].
type formPair [2]string

// formValues is a full slice of all the key/value pairs from a form-encoded
// string.
type formValues []formPair

// parseForm parses a form-encoded body or query into a set of key/value pairs.
// It differs from url.ParseQuery in that because it produces a slice instead
// of a map, order can be preserved. This is key to properly decoding
// "Rack-style" form encoding.
//
// Implementation modified from: https://github.com/deoxxa/urlqp
func parseForm(s string) (formValues, error) {
	s = strings.TrimPrefix(s, "?")

	if s == "" {
		return nil, nil
	}

	rawValues := strings.Split(s, "&")
	r := make(formValues, len(rawValues))

	for i, rawValue := range rawValues {
		// Split this raw form value into two parts, at the first `=`
		valueParts := strings.SplitN(rawValue, "=", 2)

		formKey, err := url.QueryUnescape(valueParts[0])
		if err != nil {
			return nil, err
		}

		// Set a default for the value. Empty seems reasonable.
		v := ""

		// If `b` has more than one element, that means the second one will be the
		// parameter value, so grab it.
		if len(valueParts) > 1 {
			v, err = url.QueryUnescape(valueParts[1])
			if err != nil {
				return nil, err
			}
		}

		r[i] = formPair{formKey, v}
	}

	return r, nil
}

func parseFormValues(form formValues) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	for _, pair := range form {
		key := pair[0]
		value := pair[1]

		keyParts := parseKey(key)

		if len(keyParts) == 0 {
			continue
		}

		rawkeyPart := keyParts[0]
		if rawkeyPart.KeyType() != keyTypeRaw {
			return nil, fmt.Errorf(`Invalid key "%v". Keys must start with a name.`, key)
		}

		pairParams, err := buildParamStructure(rawkeyPart.Content(), keyParts[1:], value)
		if err != nil {
			return nil, err
		}

		//fmt.Printf("new params = %+v\n", pairParams)
		mergeMapsRecursive(params, pairParams)
		//fmt.Printf("merge result = %+v\n\n", params)
	}

	return params, nil
}
