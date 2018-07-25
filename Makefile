all: test vet lint check-gofmt

check-gofmt:
	scripts/check_gofmt.sh

# This command is overcomplicated because Golint's `./...` doesn't filter
# `vendor/` (unlike every other Go command).
lint:
	go list ./... | xargs -I{} -n1 sh -c 'golint -set_exit_status {} || exit 255'

test:
	go test ./...

vet:
	go vet ./...
