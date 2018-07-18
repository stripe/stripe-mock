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

const defaultPortHTTP = 12111
const defaultPortHTTPS = 12112

// verbose tracks whether the program is operating in verbose mode
var verbose bool

// This is set to the actual version by GoReleaser (using `-ldflags "-X ..."`)
// as it's run. Versions built from source will always show master.
var version = "master"

// ---

func main() {
	var options options

	flag.BoolVar(&options.http, "http", false, "Run with HTTP")
	flag.IntVar(&options.httpPort, "http-port", 0, "Port to listen on for HTTP")
	flag.StringVar(&options.httpUnixSocket, "http-unix", "", "Unix socket to listen on for HTTP")

	flag.BoolVar(&options.https, "https", false, "Run with HTTPS (which also allows HTTP/2 to be activated)")
	flag.IntVar(&options.httpsPort, "https-port", 0, "Port to listen on for HTTPS")
	flag.StringVar(&options.httpsUnixSocket, "https-unix", "", "Unix socket to listen on for HTTPS")

	flag.IntVar(&options.port, "port", 0, "Port to listen on (also respects PORT from environment)")
	flag.StringVar(&options.fixturesPath, "fixtures", "", "Path to fixtures to use instead of bundled version")
	flag.StringVar(&options.specPath, "spec", "", "Path to OpenAPI spec to use instead of bundled version")
	flag.StringVar(&options.unixSocket, "unix", "", "Unix socket to listen on")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode")
	flag.BoolVar(&options.showVersion, "version", false, "Show version and exit")

	flag.Parse()

	fmt.Printf("stripe-mock %s\n", version)
	if options.showVersion || len(flag.Args()) == 1 && flag.Arg(0) == "version" {
		return
	}

	err := options.checkConflictingOptions()
	if err != nil {
		flag.Usage()
		abort(fmt.Sprintf("Invalid options: %v", err))
	}

	// For both spec and fixtures stripe-mock will by default load data from
	// internal assets compiled into the binary, but either one can be
	// overridden with a -spec or -fixtures argument and a path to a file.
	stripeSpec, err := getSpec(options.specPath)
	if err != nil {
		abort(err.Error())
	}

	fixtures, err := getFixtures(options.fixturesPath)
	if err != nil {
		abort(err.Error())
	}

	stub := StubServer{fixtures: fixtures, spec: stripeSpec}
	err = stub.initializeRouter()
	if err != nil {
		abort(fmt.Sprintf("Error initializing router: %v\n", err))
	}

	http.HandleFunc("/", stub.HandleRequest)

	httpListener, err := options.getHTTPListener()
	if err != nil {
		abort(err.Error())
	}

	// Only start HTTP if requested (it's the default, but it won't start if
	// HTTPS is explicitly requested instead)
	if httpListener != nil {
		server := http.Server{}

		// Listen in a new Goroutine that so we can start a simultaneous HTTPS
		// listener if necessary.
		go func() {
			err := server.Serve(httpListener)
			if err != nil {
				abort(err.Error())
			}
		}()
	}

	httpsListener, err := options.getNonSecureHTTPSListener()
	if err != nil {
		abort(err.Error())
	}

	// Only start HTTPS if requested
	if httpsListener != nil {
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
		tlsListener := tls.NewListener(httpsListener, tlsConfig)

		go func() {
			err := server.Serve(tlsListener)
			if err != nil {
				abort(err.Error())
			}
		}()
	}

	// Block forever. The serve Goroutines above will abort the program if
	// either of them fails.
	select {}
}

//
// Private types
//

// options is a container for the command line options passed to stripe-mock.
type options struct {
	fixturesPath string

	http           bool
	httpPort       int
	httpUnixSocket string

	https           bool
	httpsPort       int
	httpsUnixSocket string

	port        int
	showVersion bool
	specPath    string
	unixSocket  string
}

func (o *options) checkConflictingOptions() error {
	if o.unixSocket != "" && o.port != 0 {
		return fmt.Errorf("Please specify only one of -port or -unix")
	}

	//
	// HTTP
	//

	if o.http && (o.httpUnixSocket != "" || o.httpPort != 0) {
		return fmt.Errorf("Please don't specify -http when using -http-port or -http-unix")
	}

	if (o.unixSocket != "" || o.port != 0) && (o.httpUnixSocket != "" || o.httpPort != 0) {
		return fmt.Errorf("Please don't specify -port or -unix when using -http-port or -http-unix")
	}

	if o.httpUnixSocket != "" && o.httpPort != 0 {
		return fmt.Errorf("Please specify only one of -http-port or -http-unix")
	}

	//
	// HTTPS
	//

	if o.https && (o.httpsUnixSocket != "" || o.httpsPort != 0) {
		return fmt.Errorf("Please don't specify -https when using -https-port or -https-unix")
	}

	if (o.unixSocket != "" || o.port != 0) && (o.httpsUnixSocket != "" || o.httpsPort != 0) {
		return fmt.Errorf("Please don't specify -port or -unix when using -https-port or -https-unix")
	}

	if o.httpsUnixSocket != "" && o.httpsPort != 0 {
		return fmt.Errorf("Please specify only one of -https-port or -https-unix")
	}

	return nil
}

// getHTTPListener gets a listener on a port or unix socket depending on the
// options provided. If HTTP should not be enabled, it returns nil.
func (o *options) getHTTPListener() (net.Listener, error) {
	if o.httpPort != 0 {
		return getPortListener(o.httpPort)
	}

	if o.httpUnixSocket != "" {
		return getUnixSocketListener(o.httpUnixSocket)
	}

	// HTTP is active by default, but only if HTTPS is *not* active
	if o.https || o.httpsPort != 0 || o.httpsUnixSocket != "" {
		return nil, nil
	}

	if o.port != 0 {
		return getPortListener(o.port)
	}

	if o.unixSocket != "" {
		return getUnixSocketListener(o.unixSocket)
	}

	return getPortListenerDefault(defaultPortHTTP)
}

// getNonSecureHTTPSListener gets a basic listener on a port or unix socket
// depending on the options provided. Its return listener must still be wrapped
// in a TLSListener. If HTTPS should not be enabled, it returns nil.
func (o *options) getNonSecureHTTPSListener() (net.Listener, error) {
	if o.httpsPort != 0 {
		return getPortListener(o.httpsPort)
	}

	if o.httpsUnixSocket != "" {
		return getUnixSocketListener(o.httpsUnixSocket)
	}

	// HTTPS is disabled by default
	if !o.https {
		return nil, nil
	}

	if o.port != 0 {
		return getPortListener(o.port)
	}

	if o.unixSocket != "" {
		return getUnixSocketListener(o.unixSocket)
	}

	return getPortListenerDefault(defaultPortHTTPS)
}

//
// Private functions
//

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

func getPortListener(port int) (net.Listener, error) {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, fmt.Errorf("Error listening on port: %v\n", err)
	}

	fmt.Printf("Listening on port: %v\n", port)
	return listener, nil
}

// getPortListenerDefault gets a port listener based on the environment
// variable `PORT`, or falls back to a listener on the default port
// (`defaultPort`) if one was not present.
func getPortListenerDefault(defaultPort int) (net.Listener, error) {
	if os.Getenv("PORT") != "" {
		envPort, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			return nil, err
		}
		return getPortListener(envPort)
	}

	return getPortListener(defaultPort)
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

func getUnixSocketListener(unixSocket string) (net.Listener, error) {
	listener, err := net.Listen("unix", unixSocket)
	if err != nil {
		return nil, fmt.Errorf("Error listening on socket: %v\n", err)
	}

	fmt.Printf("Listening on Unix socket: %v\n", unixSocket)
	return listener, nil
}
