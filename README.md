# stripe-mock [![Build Status](https://travis-ci.org/stripe/stripe-mock.svg?branch=master)](https://travis-ci.org/stripe/stripe-mock)

stripe-mock is a mock HTTP server that responds like the real Stripe API. It
can be used instead of Stripe's testmode to make test suites integrating with
Stripe faster and less brittle.

stripe-mock is powered by [the Stripe OpenAPI specification][openapi], which is
generated from within the backend API implementation. It operates statelessly
(i.e. it won't remember new resources that are created with it) and responds
with sample data that's generated using a similar scheme to the one found [in
the API reference][apiref].

## Usage

If you have Go installed, you can install the basic binary with:

``` sh
go get -u github.com/stripe/stripe-mock
```

With no arguments, stripe-mock will listen with HTTP on its default port of
`12111`:

``` sh
stripe-mock
```

It can also be activated with HTTPS (and by extension support for HTTP/2) using
the `-https` flag (the default port changes to `12112` for HTTPS):

``` sh
stripe-mock -https
```

For either HTTP or HTTPS, the port can be specified with either the `PORT`
environmental variable or the `-port` option (the latter is preferred if both
are present):

``` sh
stripe-mock -port 12111
```

It can also listen on a Unix socket:

``` sh
stripe-mock -https -unix /tmp/stripe-mock-secure.sock
```

It can be configured to receive both HTTP _and_ HTTPS by using the
`-http-port`, `-http-unix`, `-https-port`, and `-https-unix` options (and note
that these cannot be mixed with any of the basic options above):

``` sh
stripe-mock -http-port 12111 -https-port 12112
```

### Homebrew

Get it from Homebrew or download it [from the releases page][releases]:

``` sh
brew install stripe/stripe-mock/stripe-mock

# start a stripe-mock service at login
brew services start stripe-mock

# upgrade if you already have it
brew upgrade stripe-mock
```

The Homebrew service listens on port `12111` for HTTP and `12112` for HTTPS and
HTTP/2.

### Docker

``` sh
# build
docker build . -t stripe-mock
# run
docker run -p 12111:12112 stripe-mock
```

The default Docker `ENTRYPOINT` listens on port `12111` for HTTP and `12112`
for HTTPS and HTTP/2.

### Sample request

After you've started stripe-mock, you can try a sample request against it:

``` sh
curl -i http://localhost:12111/v1/charges -H "Authorization: Bearer sk_test_123"
```

## Development

### Testing

Run the test suite:

``` sh
go test ./...
```

### Binary data & updating OpenAPI

The project uses [go-bindata] to bundle OpenAPI and fixture data into
`bindata.go` so that it's automatically included with built executables.
Rebuild it with:

``` sh
# Make sure you have the go-bindata executable (it's not vendored into this
# repository).
go get -u github.com/jteeuwen/go-bindata/...

# Drop into the openapi/ Git submodule and update it (you may have to commit a
# change).
pushd openapi/ && git pull origin master && popd

# Generates `bindata.go`.
go generate
```

## Release

Release builds are generated with [goreleaser]. Make sure you have the software
and a `GITHUB_TOKEN`:

``` sh
go get -u github.com/goreleaser/goreleaser
export GITHUB_TOKEN=...
```

Commit changes and tag `HEAD`:

``` sh
git pull origin --tags
git tag v0.1.1
git push origin --tags
```

Then run goreleaser and you're done! Check [releases] (it also pushes to the
Homebrew tap).

``` sh
goreleaser --rm-dist
```

[apiref]: https://stripe.com/docs/api
[go-bindata]: https://github.com/jteeuwen/go-bindata
[goreleaser]: https://github.com/goreleaser/goreleaser
[openapi]: https://github.com/stripe/openapi
[releases]: https://github.com/stripe/stripe-mock/releases

<!--
# vim: set tw=79:
-->
