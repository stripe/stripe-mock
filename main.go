//go:generate go-bindata cert/cert.pem cert/key.pem openapi/openapi/fixtures3.json openapi/openapi/spec3.json

package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/stripe/stripe-mock/spec"
	"gopkg.in/yaml.v2"
)

const defaultPort = 12111

// verbose tracks whether the program is operating in verbose mode
var verbose bool

// This is set to the actual version by GoReleaser (using `-ldflags "-X ..."`)
// as it's run. Versions built from source will always show master.
var version = "master"

// ---

func main() {
	var https bool
	var showVersion bool
	var port int
	var fixturesPath string
	var specPath string
	var unix string

	flag.BoolVar(&https, "https", false, "Run with HTTPS (which also allows HTTP/2 to be activated)")
	flag.IntVar(&port, "port", 0, "Port to listen on (also respects PORT from environment)")
	flag.StringVar(&fixturesPath, "fixtures", "", "Path to fixtures to use instead of bundled version")
	flag.StringVar(&specPath, "spec", "", "Path to OpenAPI spec to use instead of bundled version")
	flag.StringVar(&unix, "unix", "", "Unix socket to listen on")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode")
	flag.BoolVar(&showVersion, "version", false, "Show version and exit")
	flag.Parse()

	fmt.Printf("stripe-mock %s\n", version)
	if showVersion || len(flag.Args()) == 1 && flag.Arg(0) == "version" {
		return
	}

	if unix != "" && port != 0 {
		flag.Usage()
		abort("Specify only one of -port or -unix\n")
	}

	// For both spec and fixtures stripe-mock will by default load data from
	// internal assets compiled into the binary, but either one can be
	// overridden with a -spec or -fixtures argument and a path to a file.
	stripeSpec, err := getSpec(specPath)
	if err != nil {
		abort(err.Error())
	}

	fixtures, err := getFixtures(fixturesPath)
	if err != nil {
		abort(err.Error())
	}

	stub := StubServer{fixtures: fixtures, spec: stripeSpec}
	err = stub.initializeRouter()
	if err != nil {
		abort(fmt.Sprintf("Error initializing router: %v\n", err))
	}

	http.HandleFunc("/", stub.HandleRequest)

	listener, err := getListener(port, unix)
	if err != nil {
		abort(err.Error())
	}

	if https {
		// Our self-signed certificate is bundled up using go-bindata so that
		// it stays easy to distribute stripe-mock as a standalone binary with
		// no other dependencies.
		certificate, err := getTLSCertificate()
		if err != nil {
			abort(err.Error())
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{certificate},

			// h2 is HTTP/2. A server with a default config normally doesn't
			// need this hint, but Go is somewhat inflexible, and we need this
			// here because we're using `Serve` and reading a TLS certificate
			// from memory instead of using `ServeTLS` which would've read a
			// certificate from file.
			NextProtos: []string{"h2"},
		}

		server := http.Server{TLSConfig: tlsConfig}
		tlsListener := tls.NewListener(listener, tlsConfig)
		server.Serve(tlsListener)
	} else {
		server := http.Server{}
		server.Serve(listener)
	}
}

// ---

func abort(message string) {
	fmt.Fprintf(os.Stderr, message)
	os.Exit(1)
}

// getTLSCertificate reads a certificate and key from the assets built by
// go-bindata.
func getTLSCertificate() (tls.Certificate, error) {
	cert, err := Asset("cert/cert.pem")
	if err != nil {
		return tls.Certificate{}, err
	}

	key, err := Asset("cert/key.pem")
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(cert, key)
}

// getEnvPortOrDefault gets a port from the environment variable `PORT` or
// falls back to the default port (`defaultPort`) if one was not present.
func getEnvPortOrDefault() (int, error) {
	if os.Getenv("PORT") != "" {
		port, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			return 0, err
		}
		return port, nil
	}

	return defaultPort, nil
}

func getFixtures(fixturesPath string) (*spec.Fixtures, error) {
	var data []byte
	var err error
	var isYAML bool

	if fixturesPath == "" {
		// And do the same for fixtures
		data, err = Asset("openapi/openapi/fixtures3.json")
	} else {
		data, err = ioutil.ReadFile(fixturesPath)

		if filepath.Ext(fixturesPath) == ".yaml" {
			isYAML = true
		}
	}

	if err != nil {
		return nil, fmt.Errorf("Error loading fixtures: %v\n", err)
	}

	var fixtures spec.Fixtures

	if isYAML {
		err = yaml.Unmarshal(data, &fixtures)
		if err == nil {
			// To support boolean keys, the `yaml` package unmarshals maps to
			// map[interface{}]interface{}. Here we recurse through the result
			// and change all maps to map[string]interface{} like we would've
			// gotten from `json`.
			for k, v := range fixtures.Resources {
				fixtures.Resources[k] = stringifyKeysMapValue(v)
			}
		}
	} else {
		err = json.Unmarshal(data, &fixtures)
	}

	if err != nil {
		return nil, fmt.Errorf("Error decoding spec: %v\n", err)
	}
	return &fixtures, nil
}

func getListener(port int, unix string) (net.Listener, error) {
	var err error
	var listener net.Listener

	if unix != "" {
		listener, err = net.Listen("unix", unix)
		fmt.Printf("Listening on unix socket %v\n", unix)
	} else {
		if port == 0 {
			port, err = getEnvPortOrDefault()
			if err != nil {
				return nil, err
			}
		}
		listener, err = net.Listen("tcp", ":"+strconv.Itoa(port))
		fmt.Printf("Listening on port %v\n", port)
	}
	if err != nil {
		return nil, fmt.Errorf("Error listening on socket: %v\n", err)
	}
	return listener, nil
}

func getSpec(specPath string) (*spec.Spec, error) {
	var data []byte
	var err error
	var isYAML bool

	if specPath == "" {
		// Load the spec information from go-bindata
		data, err = Asset("openapi/openapi/spec3.json")
	} else {
		data, err = ioutil.ReadFile(specPath)

		if filepath.Ext(specPath) == ".yaml" {
			isYAML = true
		}
	}
	if err != nil {
		return nil, fmt.Errorf("Error loading spec: %v\n", err)
	}

	var stripeSpec spec.Spec

	if isYAML {
		err = yaml.Unmarshal(data, &stripeSpec)
	} else {
		err = json.Unmarshal(data, &stripeSpec)
	}
	if err != nil {
		return nil, fmt.Errorf("Error decoding spec: %v\n", err)
	}
	return &stripeSpec, nil
}
