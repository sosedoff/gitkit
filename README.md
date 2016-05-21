# gitkit

Smart HTTP git server for Go

## Install

```
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

```
go run example.go
```

## Sources

This code was based on the following sources:

- https://github.com/flynn/flynn/tree/master/gitreceive
- https://gitlab.com/gitlab-org/gitlab-workhorse/tree/master/internal/git

## License

MIT