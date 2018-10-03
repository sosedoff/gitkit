test:
	go test -v -race -cover .

build:
	go build

all:
	gox -osarch="darwin/amd64 linux/amd64" -output="gitkit_{{.OS}}_{{.Arch}}"

setup:
	go get -u github.com/golang/dep/cmd/dep
	go get -u golang.org/x/tools/cmd/cover
	go get -u github.com/mitchellh/gox
	dep ensure