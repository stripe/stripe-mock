GIT_COMMITSHA = $(shell git rev-parse HEAD)
IMAGE_NAME = "stripe/stripe-mock"
OPENAPI_BRANCH ?= master

all: test vet lint check-gofmt build

build:
	go build -mod=vendor -o stripe-mock

check-gofmt:
	scripts/check_gofmt.sh

lint:
	staticcheck

test:
	# -count=1 disables the cache
	go test ./... -count=1

vet:
	go vet ./...

docker-build:
	docker build -t "$(IMAGE_NAME):latest" -t "$(IMAGE_NAME):$(GIT_COMMITSHA)" .
.PHONY: docker-build

docker-run:
	docker run --rm -it -p 12111-12112:12111-12112 "$(IMAGE_NAME):latest"
.PHONY: docker-run

update-openapi-spec:
	rm -f ./embedded/openapi/spec3.json
	rm -f ./embedded/openapi/fixtures3.json
	wget https://raw.githubusercontent.com/stripe/openapi/$(OPENAPI_BRANCH)/openapi/spec3.json -P ./embedded/openapi
	wget https://raw.githubusercontent.com/stripe/openapi/$(OPENAPI_BRANCH)/openapi/fixtures3.json -P ./embedded/openapi
.PHONY: update-openapi-spec
