FROM golang:1.9-alpine

ADD . /go/src/github.com/stripe/stripe-mock

RUN go install github.com/stripe/stripe-mock

EXPOSE 12111

ENTRYPOINT /go/bin/stripe-mock
