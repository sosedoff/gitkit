test:
	go test -v -race -cover .

build:
	go build

all:
	gox -osarch="darwin/amd64 linux/amd64" -output="gitkit_{{.OS}}_{{.Arch}}"

setup:
	go get github.com/tools/godep
	go get golang.org/x/tools/cmd/cover
	go get github.com/mitchellh/gox
	godep restore