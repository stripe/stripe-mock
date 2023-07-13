# stripe-mock [![Build Status](https://travis-ci.org/stripe/stripe-mock.svg?branch=master)](https://travis-ci.org/stripe/stripe-mock)

stripe-mock is a mock HTTP server based on the real Stripe API. It accepts the
same requests and parameters that the Stripe API accepts, and rejects requests
whose parameters are not recognized or have incorrect types. Its responses
resemble the responses of the real Stripe API in terms of data type; however,
stripe-mock **does not attempt to reproduce the _behavior_ of the real Stripe API
at all**. It cannot reject all invalid requests, and its responses are completely
hardcoded. They will have a correct type, but they will not necessarily be
realistic Stripe responses.

stripe-mock is meant for basic sanity checks. We use it in the test suites of
our server-side SDKs, like [stripe-ruby](https://github.com/stripe/stripe-ruby),
[stripe-go](https://github.com/stripe/stripe-go), etc, to help validate that the
SDK hits the right URL and sends the right parameters. If you have more
sophisticated testing needs, you shouldn't use stripe-mock. Always test changes
to your Stripe integration against
[testmode](https://stripe.com/docs/keys#test-live-modes). For regression test
suites, you should define your own mocks, or use a playback testing tool such as
the [VCR gem](https://github.com/vcr/vcr).

While stripe-mock is naïve, it is powered by
[the Stripe OpenAPI specification][openapi] and is therefore kept up-to-date
with the latest methods, resources, and fields.

## Features and limitations

stripe-mock supports the following features:

- It has a catalog of every API URL and their signatures. It responds to URLs
  that exist with a resource that it returns and 404s on URLs that don't exist.
- JSON Schema is used to check the validity of the parameters of incoming
  requests. Validation is comprehensive, but far from exhaustive, so don't
  expect the full barrage of checks of the live API.
- Responses are generated based off resource fixtures. They're also generated
  from within Stripe's API, and similar to the sample data available in Stripe's
  [API reference][apiref]. **They are hardcoded**, and will not necessarily
  represent realistic responses based on the parameters you input into the
  request.
- It reflects the values of valid input parameters into responses where the
  naming and type are the same. So if a charge is created with `amount=123`, a
  charge will be returned with `"amount": 123`.
- It will respond over HTTP or over HTTPS. HTTP/2 over HTTPS is available if the
  client supports it.

Limitations:

- stripe-mock is stateless. Data you send on a `POST` request will be validated,
  but it will be completely ignored beyond that. It will not be reflected on the
  response or on any future request -- unlike the real Stripe API, which stores
  the information you send it.
- For polymorphic endpoints (say one that returns either a card or a bank
  account), only a single resource type is ever returned. There's no way to
  specify which one that is.
- It's locked to the latest version of Stripe's API and doesn't support old
  versions.
- [Testing for specific responses and errors](https://stripe.com/docs/testing#cards-responses)
  is currently not supported. It will return a success response instead of the
  desired error response.

## Future plans

The scope we envision for stripe-mock has significantly narrowed since 2017 when
we first released it. Back in 2017, our vision was for stripe-mock was to return
responses that were _realistic_ as well as just having the expected types. This
has changed. We are currently **not** planning to add statefulness or more
sophisticated testing features to stripe-mock. stripe-mock will remain a tool
for basic sanity checks. If you have more sophisticated needs, you should define
your own mocks, use a playback testing tool like the
[VCR gem](https://github.com/vcr/vcr), or find a community library you trust. Be
careful, though. Always test changes to your Stripe integration against
testmode. Mock implementations of Stripe can never behave exactly at the Stripe
API does, and might differ in nuanced (and potentially dangerous) ways.

## Usage

If you have Go installed, you can install the basic binary with:

```sh
go install github.com/stripe/stripe-mock@latest
```

With no arguments, stripe-mock will listen with HTTP on its default port of
`12111` and HTTPS on `12112`:

```sh
stripe-mock
```

Ports can be specified explicitly with:

```sh
stripe-mock -http-port 12111 -https-port 12112
```

(Leave either `-http-port` or `-https-port` out to activate stripe-mock on only
one protocol.)

Have stripe-mock select a port automatically by passing `0`:

```sh
stripe-mock -http-port 0
```

It can also listen via Unix socket:

```sh
stripe-mock -http-unix /tmp/stripe-mock.sock -https-unix /tmp/stripe-mock-secure.sock
```

### Homebrew

Get it from Homebrew or download it [from the releases page][releases]:

```sh
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

```sh
docker run --rm -it -p 12111-12112:12111-12112 stripe/stripe-mock:latest
```

The default Docker `ENTRYPOINT` listens on port `12111` for HTTP and `12112` for
HTTPS and HTTP/2.

### Sample request

After you've started stripe-mock, you can try a sample request against it:

```sh
curl -i http://localhost:12111/v1/charges -H "Authorization: Bearer sk_test_123"
```

## Development

### Testing

Run the test suite:

```sh
go test ./...
```

### Updating OpenAPI

Update the OpenAPI spec by running `make update-openapi-spec` in the root of the
repo.

```sh
make update-openapi-spec
```

## Dependencies

Dependencies are managed using [go modules][gomod] and require Go 1.11+ with
`GO111MODULE=on`.

## Release

Releases are automatically published by Travis CI using [goreleaser] when a new
tag is pushed:

```sh
git pull origin --tags
git tag v0.1.1
git push origin --tags
```

[apiref]: https://stripe.com/docs/api
[go-bindata]: https://github.com/go-bindata/go-bindata
[gomod]: https://golang.org/ref/mod
[goreleaser]: https://github.com/goreleaser/goreleaser
[openapi]: https://github.com/stripe/openapi
[releases]: https://github.com/stripe/stripe-mock/releases

<!--
# vim: set tw=79:
-->
