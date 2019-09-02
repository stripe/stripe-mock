package parser

import (
	"net/url"
	"strings"

	"github.com/stripe/stripe-mock/param/form"
)

//
// Public functions
//

// ParseFormString parses a form-encoded body or query into a set of key/value
// pairs. It differs from url.ParseQuery in that because it produces a slice
// instead of a map, order can be preserved. This is key to properly decoding
// "Rack-style" form encoding.
//
// Implementation modified from: https://github.com/deoxxa/urlqp
func ParseFormString(s string) (form.Values, error) {
	s = strings.TrimPrefix(s, "?")

	if s == "" {
		return nil, nil
	}

	rawValues := strings.Split(s, "&")
	r := make(form.Values, len(rawValues))

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

		r[i] = form.Pair{formKey, v}
	}

	return r, nil
}
