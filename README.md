# gitkit

Toolkit to build Git workflows with Go

## Install

```bash
go get github.com/sosedoff/gitkit
```

## Examples

### Smart HTTP Server

```go
package main

import (
  "log"
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

  // Configure git server. Will create git repos path if it does not exist.
  // If hooks are set, it will also update all repos with new version of hook scripts.
  if err := service.Setup(); err != nil {
    log.Fatal(err)
  }

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
# remote: Hello World! <----------------- pre-receive hook
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

#### Authentication

```go
package main

import (
  "log"
  "net/http"

  "github.com/sosedoff/gitkit"
)

func main() {
  service := gitkit.New(gitkit.Config{
    Dir:        "/path/to/repos",
    AutoCreate: true,
    Auth:       true, // Turned off by default
  })

  // Here's the user-defined authentication function.
  // If return value is false or error is set, user's request will be rejected.
  // You can hook up your database/redis/cache for authentication purposes.
  service.AuthFunc = func(cred gitkit.Credential) (bool, error) {
    log.Println("user auth request:", cred.Username, cred.Password)
    return cred.Username == "hello", nil
  }

  http.Handle("/", service)
  http.ListenAndServe(":5000", nil)
}
```

When you start the server and try to clone repo, you'll see password prompt. Two
examples below illustrate both failed and succesful authentication based on the
auth code above.

```bash
$ git clone http://localhost:5000/awesome-sauce.git
# Cloning into 'awesome-sauce'...
# Username for 'http://localhost:5060': foo
# Password for 'http://foo@localhost:5060':
# fatal: Authentication failed for 'http://localhost:5060/awesome-sauce.git/'

$ git clone http://localhost:5000/awesome-sauce.git
# Cloning into 'awesome-sauce'...
# Username for 'http://localhost:5060': hello
# Password for 'http://hello@localhost:5060':
# warning: You appear to have cloned an empty repository.
# Checking connectivity... done.
```

Git also allows using `.netrc` files for authentication purposes. Open your '~/.netrc'
file and add the following line:

```
machine localtion
  login hello
  password world
```

Next time you try clone the same localhost git repo, git wont show password promt.
Keep in mind that the best practice is to use auth tokens instead of plaintext passwords
for authentication. See [Heroku's docs](https://devcenter.heroku.com/articles/authentication#api-token-storage)
for more information.

## Sources

This code was based on the following sources:

- https://github.com/flynn/flynn/tree/master/gitreceive
- https://gitlab.com/gitlab-org/gitlab-workhorse/tree/master/internal/git
- https://github.com/gogits/gogs/blob/master/modules/ssh/ssh.go

Git HTTP protocol documentation:

- https://git-scm.com/book/en/v2/Git-Internals-Transfer-Protocols

## License

MIT