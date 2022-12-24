test:
	go test -v -race -cover .

build:
	go build

lint:
	golangci-lint run

all:
	gox -osarch="darwin/amd64 linux/amd64" -output="gitkit_{{.OS}}_{{.Arch}}"

