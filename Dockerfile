# Stage[0], build the Alpine Executable
FROM golang:alpine
COPY . /go/src/github.com/stripe/stripe-mock
RUN apk update; apk add git
RUN go install github.com/stripe/stripe-mock

# Stage[1], build the reduced-size image using Stage[0]'s executable
FROM alpine
COPY --from=0 /go/bin/stripe-mock /usr/bin
ADD ./versions /versions
CMD ["stripe-mock"]

# Run older versions like this:
# docker run --rm -it stripe-mock stripe-mock -spec /versions/2015-10-16/spec3.yaml -fixtures /versions/2015-10-16/fixtures3.yaml