package datareplacer

import (
	"reflect"
)

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

	return v1Value.Type() == v2Value.Type()
}
