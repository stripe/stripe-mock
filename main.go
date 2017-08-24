//go:generate go-bindata openapi/openapi/fixtures.json openapi/openapi/spec2.json

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/stripe/stripe-mock/spec"
)

const defaultPort = 12111

// verbose tracks whether the program is operating in verbose mode
var verbose bool

// This is set to the actual version by GoReleaser (using `-ldflags "-X ..."`)
// as it's run. Versions built from source will always show master.
var version = "master"

// ---

func main() {
	var showVersion bool
	var port int
	var unix string
	flag.IntVar(&port, "port", 0, "Port to listen on")
	flag.StringVar(&unix, "unix", "", "Unix socket to listen on")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.Parse()

	if showVersion || len(flag.Args()) == 1 && flag.Arg(0) == "version" {
		fmt.Printf("%s\n", version)
		return
	}

	if unix != "" && port != 0 {
		flag.Usage()
		fmt.Fprintf(os.Stderr, "Specify only one of -port or -unix\n")
		os.Exit(1)
	}

	// Load the spec information from go-bindata
	data, err := Asset("openapi/openapi/spec2.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading spec: %v\n", err)
		os.Exit(1)
	}

	var stripeSpec spec.Spec
	err = json.Unmarshal(data, &stripeSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding spec: %v\n", err)
		os.Exit(1)
	}

	// And do the same for fixtures
	data, err = Asset("openapi/openapi/fixtures.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading fixtures: %v\n", err)
		os.Exit(1)
	}

	var fixtures spec.Fixtures
	err = json.Unmarshal(data, &fixtures)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding spec: %v\n", err)
		os.Exit(1)
	}

	stub := StubServer{fixtures: &fixtures, spec: &stripeSpec}
	err = stub.initializeRouter()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing router: %v\n", err)
		os.Exit(1)
	}

	var listener net.Listener
	if unix != "" {
		listener, err = net.Listen("unix", unix)
		fmt.Printf("Listening on unix socket %v", unix)
	} else {
		if port == 0 {
			port = defaultPort
		}
		listener, err = net.Listen("tcp", ":"+strconv.Itoa(port))
		fmt.Printf("Listening on port %v", port)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listening on socket: %v\n", err)
		os.Exit(1)
	}

	http.HandleFunc("/", stub.HandleRequest)
	server := http.Server{}
	server.Serve(listener)
}
