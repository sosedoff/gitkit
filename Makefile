test:
	go test -v

build:
	go build

all:
	gox -osarch="darwin/amd64 linux/amd64" -output="gitkit_{{.OS}}_{{.Arch}}"

setup:
	go get github.com/mitchellh/gox
	go get -t