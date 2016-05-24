build:
	go build

test:
	go get
	go test -v

all:
	gox -osarch="darwin/amd64 linux/amd64" -output="gitkit_{{.OS}}_{{.Arch}}"

setup:
	go get github.com/mitchellh/gox