package main

import (
	"fmt"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func getDefaultOptions() *options {
	return &options{
		httpPort:  -1,
		httpsPort: -1,
		port:      -1,
	}
}

func TestCheckConflictingOptions(t *testing.T) {
	//
	// Valid sets of options (not exhaustive, but included quite a few standard invocations)
	//

	{
		options := getDefaultOptions()
		options.http = true

		err := options.checkConflictingOptions()
		assert.NoError(t, err)
	}

	{
		options := getDefaultOptions()
		options.https = true

		err := options.checkConflictingOptions()
		assert.NoError(t, err)
	}

	{
		options := getDefaultOptions()
		options.https = true
		options.port = 12111

		err := options.checkConflictingOptions()
		assert.NoError(t, err)
	}

	{
		options := getDefaultOptions()
		options.httpPort = 12111
		options.httpsPort = 12111

		err := options.checkConflictingOptions()
		assert.NoError(t, err)
	}

	{
		options := getDefaultOptions()
		options.httpUnixSocket = "/tmp/stripe-mock.sock"
		options.httpsUnixSocket = "/tmp/stripe-mock-secure.sock"

		err := options.checkConflictingOptions()
		assert.NoError(t, err)
	}

	//
	// Non-specific
	//

	{
		options := getDefaultOptions()
		options.port = 12111
		options.unixSocket = "/tmp/stripe-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please specify only one of -port or -unix"), err)
	}

	//
	// HTTP
	//

	{
		options := getDefaultOptions()
		options.http = true
		options.httpPort = 12111

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -http when using -http-addr, -http-port, or -http-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.http = true
		options.httpUnixSocket = "/tmp/stripe-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -http when using -http-addr, -http-port, or -http-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.port = 12111
		options.httpPort = 12111

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -port or -unix when using -http-addr, -http-port, or -http-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.unixSocket = "/tmp/stripe-mock.sock"
		options.httpUnixSocket = "/tmp/stripe-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -port or -unix when using -http-addr, -http-port, or -http-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.httpAddr = "127.0.0.1:12111"
		options.httpUnixSocket = "/tmp/stripe-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please specify only one of -http-addr, -http-port, or -http-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.httpPort = 12111
		options.httpUnixSocket = "/tmp/stripe-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please specify only one of -http-addr, -http-port, or -http-unix"), err)
	}

	//
	// HTTPS
	//

	{
		options := getDefaultOptions()
		options.https = true
		options.httpsPort = 12111

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -https when using -https-addr, -https-port, or -https-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.https = true
		options.httpsUnixSocket = "/tmp/stripe-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -https when using -https-addr, -https-port, or -https-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.port = 12111
		options.httpsPort = 12111

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -port or -unix when using -https-addr, -https-port, or -https-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.unixSocket = "/tmp/stripe-mock.sock"
		options.httpsUnixSocket = "/tmp/stripe-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -port or -unix when using -https-addr, -https-port, or -https-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.httpsAddr = "127.0.0.1:12111"
		options.httpsUnixSocket = "/tmp/stripe-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please specify only one of -https-addr, -https-port, or -https-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.httpsPort = 12111
		options.httpsUnixSocket = "/tmp/stripe-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please specify only one of -https-addr, -https-port, or -https-unix"), err)
	}
}

// Specify :0 to ask the OS for a free port.
const freePort = 0

func TestOptionsGetHTTPListener(t *testing.T) {
	// Gets a listener when explicitly requested with `-http-addr`.
	{
		options := &options{
			httpAddr: fmt.Sprintf(":%v", freePort),
		}
		listener, err := options.getHTTPListener()
		assert.NoError(t, err)
		assert.NotNil(t, listener)
		listener.Close()
	}

	// Gets a listener when explicitly requested with `-http-port`.
	{
		options := &options{
			httpPort: freePort,
		}
		listener, err := options.getHTTPListener()
		assert.NoError(t, err)
		assert.NotNil(t, listener)
		listener.Close()
	}

	// No listener when HTTPS is explicitly requested, but HTTP is not.
	{
		options := &options{
			httpPort:  -1, // Signals not specified
			httpsPort: freePort,
		}
		listener, err := options.getHTTPListener()
		assert.NoError(t, err)
		assert.Nil(t, listener)
	}

	// Activates on the default HTTP port if no other args provided.
	{
		options := &options{
			httpPortDefault: freePort,
		}
		listener, err := options.getHTTPListener()
		assert.NoError(t, err)
		assert.NotNil(t, listener)
		listener.Close()
	}
}

func TestOptionsGetNonSecureHTTPSListener(t *testing.T) {
	// Gets a listener when explicitly requested with `-https-addr`.
	{
		options := &options{
			httpsAddr: fmt.Sprintf(":%v", freePort),
		}
		listener, err := options.getNonSecureHTTPSListener()
		assert.NoError(t, err)
		assert.NotNil(t, listener)
		listener.Close()
	}

	// Gets a listener when explicitly requested with `-https-port`.
	{
		options := &options{
			httpsPort: freePort,
		}
		listener, err := options.getNonSecureHTTPSListener()
		assert.NoError(t, err)
		assert.NotNil(t, listener)
		listener.Close()
	}

	// No listener when HTTP is explicitly requested, but HTTPS is not.
	{
		options := &options{
			httpPort:  freePort,
			httpsPort: -1, // Signals not specified
		}
		listener, err := options.getNonSecureHTTPSListener()
		assert.NoError(t, err)
		assert.Nil(t, listener)
	}

	// No listener when HTTP is explicitly requested with the old `-port`
	// option.
	{
		options := &options{
			httpsPort: -1, // Signals not specified
			port:      freePort,
		}
		listener, err := options.getNonSecureHTTPSListener()
		assert.NoError(t, err)
		assert.Nil(t, listener)
	}

	// Activates on the default HTTPS port if no other args provided.
	{
		options := &options{
			httpsPortDefault: freePort,
		}
		listener, err := options.getNonSecureHTTPSListener()
		assert.NoError(t, err)
		assert.NotNil(t, listener)
		listener.Close()
	}
}
