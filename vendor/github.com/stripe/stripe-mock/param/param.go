package param

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/stripe/stripe-mock/param/form"
	"github.com/stripe/stripe-mock/param/nestedtypeassembler"
	"github.com/stripe/stripe-mock/param/parser"
)

//
// Public functions
//

// ParseParams extracts parameters from a request that an application can
// consume.
//
// Depending on the type of request, parameters may be extracted from either
// the query string, a form-encoded body, or a multipart form-encoded body (the
// latter being specific to only a very small number of endpoints).
//
// Regardless of origin, parameters are assumed to follow "Rack-style"
// conventions for encoding complex types like arrays and maps, which is how
// the Stripe API decodes data. These complex types are what makes the param
// package's implementation non-trivial. We rely on the nestedtypeassembler
// subpackage to do the heavy lifting for that.
func ParseParams(r *http.Request) (map[string]interface{}, error) {
	var values form.Values

	contentType := r.Header.Get("Content-Type")

	// Truncate content type parameters. For example, given:
	//
	//     application/json; charset=utf-8
	//
	// We want to chop off the `; charset=utf-8` at the end.
	contentType = strings.Split(contentType, ";")[0]

	formString := r.URL.RawQuery

	// Ideally we'd just parse the query string for `GET` requests and use the
	// request body for others, but the way Stripe's API actually works is to
	// mix all these request parameters together into one big bucket, behavior
	// inherited from Rack.
	//
	// Here we parse the query string regardless of the request type, then move
	// on to potentially parse the body if it looks like we should.
	var err error
	values, err = parser.ParseFormString(formString)
	if err != nil {
		return nil, err
	}

	if contentType == multipartMediaType {
		err := r.ParseMultipartForm(maxMemory)
		if err != nil {
			return nil, err
		}

		for key, keyValues := range r.MultipartForm.Value {
			for _, keyValue := range keyValues {
				values = append(values, form.Pair{key, keyValue})
			}
		}

		for key, keyValues := range r.MultipartForm.File {
			for _, keyFileHeader := range keyValues {
				file, err := keyFileHeader.Open()
				if err != nil {
					return nil, err
				}

				keyFileBytes, err := ioutil.ReadAll(file)
				file.Close()
				if err != nil {
					return nil, err
				}

				values = append(values, form.Pair{key, string(keyFileBytes)})
			}
		}
	} else if r.Method != "GET" {
		formBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		r.Body.Close()

		formString := string(formBytes)

		formValues, err := parser.ParseFormString(formString)
		if err != nil {
			return nil, err
		}

		values = append(values, formValues...)
	}

	return nestedtypeassembler.AssembleParams(values)
}

//
// Private constants
//

// maxMemory is the maximum amount of memory allowed when ingesting a multipart
// form.
//
// Set to 1 MB.
const maxMemory = 1 * 1024 * 1024

// multipartMediaType is the `Content-Type` for a multipart request.
const multipartMediaType = "multipart/form-data"
