package param

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestParseParams_Get(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?query_param=query_val", nil)
	params, err := ParseParams(req)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"query_param": "query_val",
	}, params)
}

func TestParseParams_Form(t *testing.T) {
	// Basic form in the request body.
	{
		req := httptest.NewRequest(http.MethodPost, "/",
			bytes.NewBufferString("body_param=body_val"))
		params, err := ParseParams(req)
		assert.NoError(t, err)
		assert.Equal(t, map[string]interface{}{
			"body_param": "body_val",
		}, params)
	}

	// Requests with a form body should also include values from the query
	// string, if any were sent.
	{
		req := httptest.NewRequest(http.MethodPost, "/?query_param=query_val",
			bytes.NewBufferString("body_param=body_val"))
		params, err := ParseParams(req)
		assert.NoError(t, err)
		assert.Equal(t, map[string]interface{}{
			"body_param":  "body_val",
			"query_param": "query_val",
		}, params)
	}
}

func TestParseParams_MultipartForm(t *testing.T) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	assert.NoError(t, w.WriteField("foo", "bar"))
	assert.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	params, err := ParseParams(req)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"foo": "bar",
	}, params)
}

// Differs from the above by sending a value as a file instead of as just a
// basic multipart parameter. This more accurately represents what a file
// upload would look like.
func TestParseParams_MultipartFormFromFile(t *testing.T) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fieldW, err := w.CreateFormFile("foo", "foo.txt")
	assert.NoError(t, err)
	_, err = fieldW.Write([]byte("bar"))
	assert.NoError(t, err)
	assert.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Type", w.FormDataContentType())

	params, err := ParseParams(req)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"foo": "bar",
	}, params)
}
