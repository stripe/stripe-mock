# stripe-mock [![Build Status](https://travis-ci.org/stripe/stripe-mock.svg?branch=master)](https://travis-ci.org/stripe/stripe-mock)

stripe-mock is a mock HTTP server that responds like the real Stripe API. It
can be used instead of Stripe's test mode to make test suites integrating with
Stripe faster and less brittle. It's powered by [the Stripe OpenAPI
specification][openapi], which is generated from within Stripe's API.

## Current state of development

stripe-mock is able to generate an approximately correct API response for any
endpoint, but the logic for doing so is still quite naive. It supports the
following features:

* It has a catalog of every API URL and their signatures. It responds on URLs
  that exist with a resource that it returns and 404s on URLs that don't exist.
* JSON Schema is used to check the validity of the parameters of incoming
  requests. Validation is comprehensive, but far from exhaustive, so don't
  expect the full barrage of checks of the live API.
* Responses are generated based off resource fixtures. They're also generated
  from within Stripe's API, and similar to the sample data available in
  Stripe's [API reference][apiref].
* It reflects the values of valid input parameters into responses where the
  naming and type are the same. So if a charge is created with `amount=123`, a
  charge will be returned with `"amount": 123`.
* It will respond over HTTP or over HTTPS. HTTP/2 over HTTPS is available if
  the client supports it.

Limitations:

* It's currently stateless. Data created with `POST` calls won't be stored so
  that the same information is available later.
* For polymorphic endpoints (say one that returns either a card or a bank
  account), only a single resource type is ever returned. There's no way to
  specify which one that is.
* It's locked to the latest version of Stripe's API and doesn't support old
  versions.

## Future plans

The next important feature that we're aiming to provide is statefulness. The
idea would be that resources created during a session would be stored for that
session's duration and could be subsequently retrieved, updated, and deleted.
This would allow more comprehensive integration tests to run successfully
against stripe-mock.

We'll continue to aim to improve the quality of stripe-mock's responses, but it
will never be on perfect parity with the live API. We think the ideal test
suite for an integration would involve running most of the suite against
stripe-mock, and then to have a few smoke tests run critical flows against the
more accurate (but also slower) Stripe API in test mode.

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
docker run -p 12111-12112:12111-12112 stripe-mock
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
