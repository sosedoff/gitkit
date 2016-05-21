# gitkit

Smart HTTP git server for Go

## Install

```bash
go get github.com/sosedoff/gitkit
```

## Example

```go
package main

import (
  "net/http"
  "github.com/sosedoff/gitkit"
)

func main() {
  service := gitkit.New(gitkit.Config{
    Dir:        "/path/to/repos",
    AutoCreate: true,
    Hooks: map[string][]byte{
      "pre-receive": []byte(`echo "Hello World!"`),
    },
  })

  http.Handle("/", service)
  http.ListenAndServe(":5000", nil)
}
```

Run example:

```bash
go run example.go
```

Then try to clone a test repository:

```bash
$ git clone http://localhost:5000/test.git /tmp/test
# Cloning into '/tmp/test'...
# warning: You appear to have cloned an empty repository.
# Checking connectivity... done.

$ cd /tmp/test
$ touch sample

$ git add sample
$ git commit -am "First commit"
# [master (root-commit) fe40c98] First commit
# 1 file changed, 0 insertions(+), 0 deletions(-)
# create mode 100644 sample

$ git push origin master
# Counting objects: 3, done.
# Writing objects: 100% (3/3), 213 bytes | 0 bytes/s, done.
# Total 3 (delta 0), reused 0 (delta 0)
# remote: Hello World!
# To http://localhost:5060/test.git
# * [new branch]      master -> master
```

In the example's console you'll see something like this:

```bash
2016/05/20 20:01:42 request: GET localhost:5000/test.git/info/refs?service=git-upload-pack
2016/05/20 20:01:42 repo-init: creating pre-receive hook for test.git
2016/05/20 20:03:34 request: GET localhost:5000/test.git/info/refs?service=git-receive-pack
2016/05/20 20:03:34 request: POST localhost:5000/test.git/git-receive-pack
```

## Sources

This code was based on the following sources:

- https://github.com/flynn/flynn/tree/master/gitreceive
- https://gitlab.com/gitlab-org/gitlab-workhorse/tree/master/internal/git

Git HTTP protocol documentation:

- https://git-scm.com/book/en/v2/Git-Internals-Transfer-Protocols

## License

MIT