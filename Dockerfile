# -*- mode: dockerfile -*-
#
# A multi-stage Dockerfile that builds a Linux target then creates a small
# final image for deployment.

#
# STAGE 1
#
# Uses a Go image to build a release binary.
#

FROM golang:1.9-alpine AS builder
WORKDIR /go/src/github.com/stripe/stripe-mock/
ADD ./ ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o stripe-mock .

#
# STAGE 2
#
# Use a tiny base image (alpine) and copy in the release target. This produces
# a very small output image for deployment.
#

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /go/src/github.com/stripe/stripe-mock/stripe-mock .
ENTRYPOINT ["/stripe-mock", "-http-port", "12111", "-https-port", "12112"]
EXPOSE 12111
EXPOSE 12112
