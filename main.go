package main

import (
	"crypto/tls"
	_ "embed"
	"flag"
	"fmt"
	"github.com/stripe/stripe-mock/embedded"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/stripe/stripe-mock/server"
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
	options := options{
		httpPortDefault:  defaultPortHTTP,
		httpsPortDefault: defaultPortHTTPS,
	}

	// As you can probably tell, there are just too many HTTP/HTTPS binding
	// options, which is a result of me not thinking through the original
	// interface well enough.
	//
	// I've left them all in place for now for backwards compatibility, but we
	// should probably deprecate `-http-port`, `-https-port`, `-port`, and
	// `-unix` in favor of the remaining more expressive and more versatile
	// alternatives.
	//
	// Eventually, `-http` and `-https` could become shorthand synonyms for
	// `-http-addr` and `-https-addr`.
	flag.BoolVar(&options.http, "http", false, "Run with HTTP")
	flag.StringVar(&options.httpAddr, "http-addr", "", fmt.Sprintf("Host and port to listen on for HTTP as `<ip>:<port>`; empty <ip> to bind all system IPs, empty <port> to have system choose; e.g. ':%v', '127.0.0.1:%v'", defaultPortHTTP, defaultPortHTTP))
	flag.IntVar(&options.httpPort, "http-port", -1, "Port to listen on for HTTP; same as '-http-addr :<port>'")
	flag.StringVar(&options.httpUnixSocket, "http-unix", "", "Unix socket to listen on for HTTP")

	flag.BoolVar(&options.https, "https", false, "Run with HTTPS; also enables HTTP/2")
	flag.StringVar(&options.httpsAddr, "https-addr", "", fmt.Sprintf("Host and port to listen on for HTTPS as `<ip>:<port>`; empty <ip> to bind all system IPs, empty <port> to have system choose; e.g. ':%v', '127.0.0.1:%v'", defaultPortHTTPS, defaultPortHTTPS))
	flag.IntVar(&options.httpsPort, "https-port", -1, "Port to listen on for HTTPS; same as '-https-addr :<port>'")
	flag.StringVar(&options.httpsUnixSocket, "https-unix", "", "Unix socket to listen on for HTTPS")

	flag.IntVar(&options.port, "port", -1, "Port to listen on; also respects PORT from environment")
	flag.StringVar(&options.fixturesPath, "fixtures", "", "Path to fixtures to use instead of bundled version (should be JSON)")
	flag.StringVar(&options.specPath, "spec", "", "Path to OpenAPI spec to use instead of bundled version (should be JSON)")
	flag.BoolVar(&options.strictVersionCheck, "strict-version-check", false, "Errors if version sent in Stripe-Version doesn't match the one in OpenAPI")
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

	server.Version = version

	// For both spec and fixtures stripe-mock will by default load data from
	// internal assets compiled into the binary, but either one can be
	// overridden with a -spec or -fixtures argument and a path to a file.
	stripeSpec, err := server.LoadSpec(embedded.OpenAPISpec, options.specPath)
	if err != nil {
		abort(err.Error())
	}

	fixtures, err := server.LoadFixtures(embedded.OpenAPIFixtures, options.fixturesPath)
	if err != nil {
		abort(err.Error())
	}

	stub, err := server.NewStubServer(fixtures, stripeSpec, options.strictVersionCheck, verbose)
	if err != nil {
		abort(fmt.Sprintf("Error initializing router: %v\n", err))
	}

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/", stub.HandleRequest)

	// Deduplicates doubled slashes in paths. e.g. `//v1/charges` becomes
	// `/v1/charges`.
	handler := &server.DoubleSlashFixHandler{Mux: httpMux}

	httpListener, err := options.getHTTPListener()
	if err != nil {
		abort(err.Error())
	}

	// Only start HTTP if requested (it will activate by default with no arguments, but it won't start if
	// HTTPS is explicitly requested and HTTP is not).
	if httpListener != nil {
		server := http.Server{
			Handler: handler,
		}

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

	// Only start HTTPS if requested (it will activate by default with no
	// arguments, but it won't start if HTTP is explicitly requested and HTTPS
	// is not).
	if httpsListener != nil {
		// Our self-signed certificate is bundled up using go:embed so that
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

		server := http.Server{
			Handler:   handler,
			TLSConfig: tlsConfig,
		}
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

	http            bool
	httpAddr        string
	httpPortDefault int // For testability -- in practice always defaultPortHTTP
	httpPort        int
	httpUnixSocket  string

	https            bool
	httpsAddr        string
	httpsPortDefault int // For testability -- in practice always defaultPortHTTPS
	httpsPort        int
	httpsUnixSocket  string

	port               int
	showVersion        bool
	specPath           string
	strictVersionCheck bool
	unixSocket         string
}

func (o *options) checkConflictingOptions() error {
	if o.unixSocket != "" && o.port != -1 {
		return fmt.Errorf("Please specify only one of -port or -unix")
	}

	//
	// HTTP
	//

	if o.http && (o.httpUnixSocket != "" || o.httpAddr != "" || o.httpPort != -1) {
		return fmt.Errorf("Please don't specify -http when using -http-addr, -http-port, or -http-unix")
	}

	if (o.unixSocket != "" || o.port != -1) && (o.httpUnixSocket != "" || o.httpAddr != "" || o.httpPort != -1) {
		return fmt.Errorf("Please don't specify -port or -unix when using -http-addr, -http-port, or -http-unix")
	}

	var numHTTPOptions int

	if o.httpUnixSocket != "" {
		numHTTPOptions++
	}
	if o.httpAddr != "" {
		numHTTPOptions++
	}
	if o.httpPort != -1 {
		numHTTPOptions++
	}

	if numHTTPOptions > 1 {
		return fmt.Errorf("Please specify only one of -http-addr, -http-port, or -http-unix")
	}

	//
	// HTTPS
	//

	if o.https && (o.httpsUnixSocket != "" || o.httpsAddr != "" || o.httpsPort != -1) {
		return fmt.Errorf("Please don't specify -https when using -https-addr, -https-port, or -https-unix")
	}

	if (o.unixSocket != "" || o.port != -1) && (o.httpsUnixSocket != "" || o.httpAddr != "" || o.httpsPort != -1) {
		return fmt.Errorf("Please don't specify -port or -unix when using -https-addr, -https-port, or -https-unix")
	}

	var numHTTPSOptions int

	if o.httpsUnixSocket != "" {
		numHTTPSOptions++
	}
	if o.httpsAddr != "" {
		numHTTPSOptions++
	}
	if o.httpsPort != -1 {
		numHTTPSOptions++
	}

	if numHTTPSOptions > 1 {
		return fmt.Errorf("Please specify only one of -https-addr, -https-port, or -https-unix")
	}

	return nil
}

// getHTTPListener gets a listener on a port or unix socket depending on the
// options provided. If HTTP should not be enabled, it returns nil.
func (o *options) getHTTPListener() (net.Listener, error) {
	protocol := "HTTP"

	if o.httpAddr != "" {
		return getPortListener(o.httpAddr, protocol)
	}

	if o.httpPort != -1 {
		return getPortListener(fmt.Sprintf(":%v", o.httpPort), protocol)
	}

	if o.httpUnixSocket != "" {
		return getUnixSocketListener(o.httpUnixSocket, protocol)
	}

	// HTTPS is active by default, but only if HTTP has not been explicitly
	// activated.
	if o.https || o.httpsPort != -1 || o.httpsUnixSocket != "" {
		return nil, nil
	}

	if o.port != -1 {
		return getPortListener(fmt.Sprintf(":%v", o.port), protocol)
	}

	if o.unixSocket != "" {
		return getUnixSocketListener(o.unixSocket, protocol)
	}

	return getPortListenerDefault(o.httpPortDefault, protocol)
}

// getNonSecureHTTPSListener gets a basic listener on a port or unix socket
// depending on the options provided. Its return listener must still be wrapped
// in a TLSListener. If HTTPS should not be enabled, it returns nil.
func (o *options) getNonSecureHTTPSListener() (net.Listener, error) {
	protocol := "HTTPS"

	if o.httpsAddr != "" {
		return getPortListener(o.httpsAddr, protocol)
	}

	if o.httpsPort != -1 {
		return getPortListener(fmt.Sprintf(":%v", o.httpsPort), protocol)
	}

	if o.httpsUnixSocket != "" {
		return getUnixSocketListener(o.httpsUnixSocket, protocol)
	}

	// HTTPS is active by default, but only if HTTP has not been explicitly
	// activated. HTTP may be activated with `-http`, `-http-port`, or
	// `-http-unix`, but also with the old backwards compatible basic `-port`
	// option.
	if o.http || o.httpPort != -1 || o.httpUnixSocket != "" || o.port != -1 {
		return nil, nil
	}

	if o.port != -1 {
		return getPortListener(fmt.Sprintf(":%v", o.port), protocol)
	}

	if o.unixSocket != "" {
		return getUnixSocketListener(o.unixSocket, protocol)
	}

	return getPortListenerDefault(o.httpsPortDefault, protocol)
}

//
// Private functions
//

func abort(message string) {
	fmt.Fprint(os.Stderr, message)
	os.Exit(1)
}

// getTLSCertificate reads a certificate and key embedded into the binary
func getTLSCertificate() (tls.Certificate, error) {
	return tls.X509KeyPair(embedded.CertCert, embedded.CertKey)
}

func getPortListener(addr string, protocol string) (net.Listener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("error listening at address: %v", err)
	}

	fmt.Printf("Listening for %s at address: %v\n", protocol, listener.Addr())
	return listener, nil
}

// getPortListenerDefault gets a port listener based on the environment
// variable `PORT`, or falls back to a listener on the default port
// (`defaultPort`) if one was not present.
func getPortListenerDefault(defaultPort int, protocol string) (net.Listener, error) {
	if os.Getenv("PORT") != "" {
		envPort, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			return nil, err
		}
		return getPortListener(fmt.Sprintf(":%v", envPort), protocol)
	}

	return getPortListener(fmt.Sprintf(":%v", defaultPort), protocol)
}

func getUnixSocketListener(unixSocket, protocol string) (net.Listener, error) {
	listener, err := net.Listen("unix", unixSocket)
	if err != nil {
		return nil, fmt.Errorf("error listening on socket: %v", err)
	}

	fmt.Printf("Listening for %s on Unix socket: %s\n", protocol, unixSocket)
	return listener, nil
}
