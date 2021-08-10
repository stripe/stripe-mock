GIT_COMMITSHA = $(shell git rev-parse HEAD)
IMAGE_NAME = "stripemock/stripe-mock"

all: test vet lint check-gofmt build

build:
	go build -mod=vendor -o stripe-mock

check-gofmt:
	scripts/check_gofmt.sh

lint:
	golint

test:
	go test ./...

vet:
	go vet ./...

docker-build:
	docker build -t "$(IMAGE_NAME):latest" -t "$(IMAGE_NAME):$(GIT_COMMITSHA)" .
.PHONY: docker-build

docker-run:
	docker run --rm -it -p 12111-12112:12111-12112 "$(IMAGE_NAME):latest"
.PHONY: docker-run
