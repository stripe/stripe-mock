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
* [Testing for specific responses and errors](https://stripe.com/docs/testing#cards-responses)
  is currently not supported. It will return a success response instead of
  the desired error response.

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
`12111` and HTTPS on `12112`:

``` sh
stripe-mock
```

Ports can be specified explicitly with:

``` sh
stripe-mock -http-port 12111 -https-port 12112
```

(Leave either `-http-port` or `-https-port` out to activate stripe-mock on only
one protocol.)

Have stripe-mock select a port automatically by passing `0`:

``` sh
stripe-mock -http-port 0
```

It can also listen via Unix socket:

``` sh
stripe-mock -http-unix /tmp/stripe-mock.sock -https-unix /tmp/stripe-mock-secure.sock
```

### Homebrew

Get it from Homebrew or download it [from the releases page][releases]:

``` sh
brew install stripe/stripe-mock/stripe-mock

# start a stripe-mock service at login
brew services start stripe-mock

# upgrade if you already have it
brew upgrade stripe-mock

# restart the service after upgrading
brew services restart stripe-mock
```

The Homebrew service listens on port `12111` for HTTP and `12112` for HTTPS and
HTTP/2.

### Docker

``` sh
docker run --rm -it -p 12111-12112:12111-12112 stripemock/stripe-mock:latest
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

## Dependencies

`vendor` was generated with the help of [dep][dep]. Generally, adding or
updating a dependency will be done with:

``` sh
dep ensure
```

More [day-to-day operations with dep here][depdaily].

Note that there is a patch in the `jsval` dependency to enrich the error
message for failures on `additionalProperties`. I'm still looking for a better
solution to this, but for now make sure that a patch like the one in 0b26185 is
applied to get the test suite passing.

## Release

Releases are automatically published by Travis CI using [goreleaser] when a
new tag is pushed:

``` sh
git pull origin --tags
git tag v0.1.1
git push origin --tags
```

[apiref]: https://stripe.com/docs/api
[dep]: https://github.com/golang/dep
[depdaily]: https://golang.github.io/dep/docs/daily-dep.html
[go-bindata]: https://github.com/jteeuwen/go-bindata
[goreleaser]: https://github.com/goreleaser/goreleaser
[openapi]: https://github.com/stripe/openapi
[releases]: https://github.com/stripe/stripe-mock/releases

<!--
# vim: set tw=79:
-->
