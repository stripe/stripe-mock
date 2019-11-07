# -*- mode: dockerfile -*-

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY stripe-mock /
ENTRYPOINT ["/stripe-mock", "-http-port", "12111", "-https-port", "12112"]
EXPOSE 12111
EXPOSE 12112
