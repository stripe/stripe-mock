//go:generate go-bindata openapi/openapi/fixtures.json openapi/openapi/spec2.json

package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
)

const defaultPort = 6065

// verbose tracks whether the program is operating in verbose mode
var verbose bool

// ---

type Fixtures struct {
	Resources map[ResourceID]interface{} `json:"resources"`
}

type HTTPVerb string

type JSONSchema struct {
	Enum       []string               `json:"enum"`
	Items      *JSONSchema            `json:"items"`
	Properties map[string]*JSONSchema `json:"properties"`
	Type       []string               `json:"type"`

	// Ref is populated if this JSON Schema is actually a JSON reference, and
	// it defines the location of the actual schema definition.
	Ref string `json:"$ref"`

	XResourceID string `json:"x-resourceId"`
}

type OpenAPIParameter struct {
	Description string      `json:"description"`
	In          string      `json:"in"`
	Name        string      `json:"name"`
	Required    bool        `json:"required"`
	Schema      *JSONSchema `json:"schema"`
}

type OpenAPIMethod struct {
	Description string                                `json:"description"`
	OperationID string                                `json:"operation_id"`
	Parameters  []OpenAPIParameter                    `json:"parameters"`
	Responses   map[OpenAPIStatusCode]OpenAPIResponse `json:"responses"`
}

type OpenAPIPath string

type OpenAPIResponse struct {
	Description string      `json:"description"`
	Schema      *JSONSchema `json:"schema"`
}

type OpenAPISpec struct {
	Definitions map[string]*JSONSchema                      `json:"definitions"`
	Paths       map[OpenAPIPath]map[HTTPVerb]*OpenAPIMethod `json:"paths"`
}

type OpenAPIStatusCode string

type ResourceID string

// ---

func main() {
	var port int
	var unix string
	flag.IntVar(&port, "port", 0, "Port to listen on")
	flag.StringVar(&unix, "unix", "", "Unix socket to listen on")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode")
	flag.Parse()

	if unix != "" && port != 0 {
		flag.Usage()
		log.Fatalf("Specify only one of -port or -unix")
	}

	// Load the spec information from go-bindata
	data, err := Asset("openapi/openapi/spec2.json")
	if err != nil {
		log.Fatalf("Error loading spec: %v", err)
	}

	var spec OpenAPISpec
	err = json.Unmarshal(data, &spec)
	if err != nil {
		log.Fatalf("Error decoding spec: %v", err)
	}

	// And do the same for fixtures
	data, err = Asset("openapi/openapi/fixtures.json")
	if err != nil {
		log.Fatalf("Error loading fixtures: %v", err)
	}

	var fixtures Fixtures
	err = json.Unmarshal(data, &fixtures)
	if err != nil {
		log.Fatalf("Error decoding spec: %v", err)
	}

	stub := StubServer{fixtures: &fixtures, spec: &spec}
	stub.initializeRouter()

	var listener net.Listener
	if unix != "" {
		listener, err = net.Listen("unix", unix)
		log.Printf("Listening on unix socket %v", unix)
	} else {
		if port == 0 {
			port = defaultPort
		}
		listener, err = net.Listen("tcp", ":"+strconv.Itoa(port))
		log.Printf("Listening on port %v", port)
	}
	if err != nil {
		log.Fatalf("Error listening on socket: %v", err)
	}

	http.HandleFunc("/", stub.handleRequest)
	server := http.Server{}
	server.Serve(listener)
}
