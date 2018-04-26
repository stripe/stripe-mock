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
	req := httptest.NewRequest(http.MethodGet, "/?foo=bar", nil)
	params, err := ParseParams(req)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"foo": "bar",
	}, params)
}

func TestParseParams_Form(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/",
		bytes.NewBufferString("foo=bar"))
	params, err := ParseParams(req)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"foo": "bar",
	}, params)
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
