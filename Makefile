# These tasks will no longer be needed after Go 1.9, whereby `vendor/` is no
# longer considered to be part of `./...`.

test:
	go test $(shell go list ./... | egrep -v '/vendor/')

vet:
	go vet $(shell go list ./... | egrep -v '/vendor/')
