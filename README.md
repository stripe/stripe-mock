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

Get it from Homebrew or download it [from the releases page][releases]:

``` sh
brew install stripe/stripe-mock/stripe-mock

# start a stripe-mock service at login
brew services start stripe-mock

# upgrade if you already have it
brew upgrade stripe-mock
```

Or if you have Go installed you can build it:

``` sh
go get -u github.com/stripe/stripe-mock
```

Run it:

``` sh
stripe-mock
```

Or with docker:
``` sh
# build
docker build . -t stripe-mock
# run
docker run -p 12111:12111 stripe-mock
```

Then from another terminal:

``` sh
curl -i http://localhost:12111/v1/charges -H "Authorization: Bearer sk_test_123"
```

By default, stripe-mock runs on port 12111, but is configurable with the
`-port` option.

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
git tag v0.1.1
git push origin --tags
```

Then run goreleaser and you're done! Check [releases] (it also pushes to the
Homebrew tap).

``` sh
goreleaser
```

[apiref]: https://stripe.com/docs/api
[go-bindata]: https://github.com/jteeuwen/go-bindata
[goreleaser]: https://github.com/goreleaser/goreleaser
[openapi]: https://github.com/stripe/openapi
[releases]: https://github.com/stripe/stripe-mock/releases

<!--
# vim: set tw=79:
-->
