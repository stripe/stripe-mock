# stripe-mock [![Build Status](https://travis-ci.org/brandur/stripe-mock.svg?branch=master)](https://travis-ci.org/brandur/stripe-mock)

stripe-mock is a mock HTTP server that responds like the real Stripe API. It can be used instead of Stripe's testmode to make test suites integrating with Stripe faster and less brittle.

Get it from Homebrew or download it [from the releases page][releases]:

``` sh
brew install brandur/stripe-mock/stripe-mock

# start a stripe-mock service at login
brew services start stripe-mock

# upgrade if you already have it
brew upgrade stripe-mock
```

Or if you have Go installed you can build it:

``` sh
go get -u github.com/brandur/stripe-mock
```

Run it:

``` sh
stripe-mock
```

Then from another terminal:

``` sh
curl -i http://localhost:12111/v1/charges
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

[go-bindata]: https://github.com/jteeuwen/go-bindata
[goreleaser]: https://github.com/goreleaser/goreleaser
[releases]: https://github.com/brandur/stripe-mock/releases

<!--
# vim: set tw=79:
-->
