package embedded

import (
	_ "embed"
)

//go:embed cert/key.pem
var CertKey []byte

//go:embed cert/cert.pem
var CertCert []byte

//go:embed openapi/fixtures3.json
var OpenAPIFixtures []byte

//go:embed openapi/spec3.json
var OpenAPISpec []byte
