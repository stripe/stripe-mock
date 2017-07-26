# stripelocal [![Build Status](https://travis-ci.org/brandur/stripelocal.svg?branch=master)](https://travis-ci.org/brandur/stripelocal)

A stub for the Stripe API powered by the OpenAPI specification that it
generates as an artifact.

Get it from Homebrew or download it [from the releases page][releases]:

``` sh
brew install brandur/stripelocal/stripelocal

# start a stripelocal service at login
brew services start stripelocal

# upgrade if you already have it
brew upgrade stripelocal
```

Or if you have Go installed you can build it:

``` sh
go get -u github.com/brandur/stripelocal
```

Run it:

``` sh
stripelocal
```

Then from another terminal:

``` sh
curl -i http://localhost:12111/v1/charges
```

By default, stripelocal runs on port 12111, but is configurable with the
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
[releases]: https://github.com/brandur/stripelocal/releases

<!--
# vim: set tw=79:
-->
