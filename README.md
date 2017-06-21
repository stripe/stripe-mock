# stripestub [![Build Status](https://travis-ci.org/brandur/stripestub.svg?branch=master)](https://travis-ci.org/brandur/stripestub)

A stub for the Stripe API based on its generated OpenAPI
specification.

``` sh
go get -u github.com/brandur/stripestub
go install && stripestub
```

Then from another terminal:

``` sh
curl -i http://localhost:6065/v1/charges
```

## Development

### Testing

Run the test suite:

``` sh
go test
```

### Binary data

The project uses [go-bindata] to bundle OpenAPI and fixture
data into `bindata.go` so that it's automatically included
with built executables. Rebuild it with:

``` sh
go get -u github.com/jteeuwen/go-bindata/...
go generate
```

[go-bindata]: https://github.com/jteeuwen/go-bindata
