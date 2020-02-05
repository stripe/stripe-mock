package datareplacer

import (
	"reflect"
)

// ReplaceData takes a generated response and replaces values in it that share
// a name and type of parameters that were sent in with the request.
//
// This is designed to have the effect of making returned fixtures more
// realistic while also staying a simple heuristic that doesn't require very
// much maintenance.
func ReplaceData(requestData map[string]interface{}, responseData map[string]interface{}) map[string]interface{} {
	for k, requestValue := range requestData {
		responseValue, ok := responseData[k]

		// Recursively call in to replace data, but only if the key is
		// in both maps.
		//
		// A fairly obvious improvement here is if a key is in the
		// request but not present in the response, then check the
		// canonical schema to see if it's there. It might be an
		// optional field that doesn't appear in the fixture, and if it
		// was given to us with the request, we probably want to
		// include it.
		if ok {
			requestKeyMap, requestKeyOK := requestValue.(map[string]interface{})
			responseKeyMap, responseKeyOK := responseValue.(map[string]interface{})

			if requestKeyOK && responseKeyOK {
				responseData[k] = ReplaceData(requestKeyMap, responseKeyMap)
			} else {
				// In the non-map case, just set the respons key's value to
				// what was in the request, but only if both values are the
				// same type (this is to prevent problems where a field is set
				// as an ID, but the response field is the hydrated object of
				// that).
				//
				// While this will largely be "good enough", there's some
				// obvious cases that aren't going to be handled correctly like
				// index-based array updates (e.g.,
				// `additional_owners[1][name]=...`). I'll have to iron out
				// that rough edges later on.
				if isSameType(requestValue, responseValue) {
					responseData[k] = requestValue
				}
			}
		}
	}

	return responseData
}

func isSameType(v1, v2 interface{}) bool {
	v1Value := reflect.ValueOf(v1)
	v2Value := reflect.ValueOf(v2)

	// Reflect in Go has the concept of a "zero Value" (not be confused with a
	// type's zero value with a lowercase "v") and asking for Type on one will
	// panic. I'm not exactly sure under what conditions these are generated,
	// but they are occasionally, so here we hedge against them.
	//
	// https://github.com/stripe/stripe-mock/issues/75
	if !v1Value.IsValid() || !v2Value.IsValid() {
		return false
	}

	v1Type := v1Value.Type()
	v2Type := v2Value.Type()

	// If we're *not* dealing with slices, we can short circuit right away by
	// just comparing types.
	if v1Type.Kind() != reflect.Slice || v2Type.Kind() != reflect.Slice {
		return v1Type == v2Type
	}


	// When working with slices we have to be a bit careful to make sure that
	// we're only replacing slices with slices of the same type because there
	// are certain endpoints that take one sort of slice and return another
	// under the same name.
	//	
	// For example, `default_tax_rates` under "create subscription" takes an
	// array of strings, but then returns an array of objects.
	//
	// Ideally we'd decide whether we can do the replacement based on what the
	// types are supposed to be as determined by OpenAPI or based on the
	// slice's type, but unfortunately this code is not set up to read OpenAPI,
	// and all types are just `[]interface{}`, so instead we have to inspect
	// the first element of each slice and determine whether we can do the
	// replacement based off whether they're the same.
	//
	// This approach is conservative in that if either slice is empty, we don't
	// have enough information to determine whether the replacement is safe.
	// This isn't ideal, but is the only decent option we have right now.
	v1Slice := v1Value.Interface().([]interface{})
	v2Slice := v2Value.Interface().([]interface{})

	if len(v1Slice) < 1 || len(v2Slice) < 1 {
		return false
	}

	return reflect.ValueOf(v1Slice[0]).Type() == reflect.ValueOf(v2Slice[0]).Type()
}
