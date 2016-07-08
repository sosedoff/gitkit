# gitkit

Toolkit to build Git workflows with Go

## Install

```bash
go get github.com/sosedoff/gitkit
```

## Smart HTTP and SSH Server

```go
package main

import (
  "github.com/sosedoff/gitkit"
)

func main() {
  service := gitkit.New()
  service.Run()
}
```
This will start a Smart HTTP server on port 8080 and an SSH server on port 2222 with the default configuration.  

**Warning!** The defaults are insecure, so take a look at the configuration section below and the `examples/` directory for advice on setting up a secure service.

Run example:

```bash
go run example.go
```

Then try to create a test repository:

```bash
$ cd /tmp/test
$ touch sample
$ git init
$ git add sample
$ git commit -am "First commit"
# [master (root-commit) fe40c98] First commit
# 1 file changed, 0 insertions(+), 0 deletions(-)
# create mode 100644 sample

$ git remote add origin http://localhost:8080/test.git
$ git push origin master
# Counting objects: 3, done.
# Writing objects: 100% (3/3), 213 bytes | 0 bytes/s, done.
# Total 3 (delta 0), reused 0 (delta 0)
# To http://localhost:8080/test.git
# * [new branch]      master -> master
```

You can also try cloning via SSH.

First put the following lines in your `~/.ssh/config` since the SSH server runs on a non-standard port:

```
Host localhost
  Port 2222
```

```bash
$ cd /tmp
$ git clone git@127.0.0.1:test.git test2
```

### HTTP Authentication

```go
package main

import (
	"crypto/subtle"
	"fmt"
	"github.com/BTBurke/gitkit"
	"log"
	"os"
)

func main() {

  // A custom authorization function
	myAuthFunc := func(c gitkit.Credential, r *gitkit.Request) (bool, error) {
		myPasswords := map[string][]byte{
			"hello": []byte("reallylongpassword"),
		}
		return subtle.ConstantTimeCompare(myPasswords[c.Username], []byte(c.Password)) == 1, nil
	}

	auth := gitkit.UseHTTPAuthFunc(myAuthFunc)
	server := gitkit.New(auth)
	server.Run()
}
```

When you start the server and try to clone repo, you'll see password prompt.

```bash
$ git clone http://localhost:8080/awesome-sauce.git
# Cloning into 'awesome-sauce'...
# Username for 'http://localhost:8080': hello
# Password for 'http://hello@localhost:8080':
```

Git also allows using `.netrc` files for authentication purposes. Open your `~/.netrc`
file and add the following line:

```
machine localhost
  login hello
  password reallylongpassword
```
or you can pass the username and password in the url like so:
```
$ git clone http://hello:reallylongpassword@127.0.0.1:8080/awesome-sauce.git
```

Next time you try clone the same localhost git repo, git wont show password promt.
Keep in mind that the best practice is to use auth tokens instead of plaintext passwords
for authentication. See [Heroku's docs](https://devcenter.heroku.com/articles/authentication#api-token-storage)
for more information.

## Configuration

Gitkit has a lot of options to configure the servers the way you want.  See the godoc and the `examples/` directory for information on configuring authorization, TLS, git hooks, and more.

To pass a custom configuration, you only need to pass options that you want to change from their default values:

```go
auth := gitkit.UseHTTPAuthFunc(myAuthFunc)
port := gitkit.HTTPPort(5000)
nossh := gitkit.EnableSSH(false)

server := gitkit.New(auth, port, nossh)
server.Run()
```
In the example above, a HTTP authorization function named `myAuthFunc` is added, the port is changed to 5000, and the SSH server is disabled.  The rest of the default configuration options are unchanged.

The default configuration is:

```go
config{
  // Git configuration
  GitPath:      "git",          // Path to git binary
  Dir:          "./",           // Repository top-level directory
  AutoCreate:   true,           // Auto create repository on first push
  AutoHooks:    false,          // Install git hooks automatically in all repos
  Hooks:        nil,            // Hooks to install in each repository
  UseNamespace: false,          // Namespaced respositories like BTBurke/gitkit  

  // HTTP server configuration
  EnableHTTP:   true,           // Enable the smart HTTP server
  HTTPAuth:     false,          // Enforce HTTP authorization
  TLSKey:       "",             // Server TLS key
  TLSCert:      "",             // Server TLS cert
  HTTPAuthFunc: nil,            // Function to use for HTTP authorization
  HTTPPort:     8080,           // HTTP service port

  // SSH server configuration
  EnableSSH:     true,          // Enable the SSH server
  KeyDir:        ".keys",       // Directory to store the server keys
  SSHKeyName:    "gitkit",      // Name of the server keys
  GitUser:       "git",         // User allowed to log in via SSH (git@yourserver:test/test.git)
  SSHAuth:       true,          // Enforce SSH authorization
  SSHAuthFunc:   nil,           // Authorization function called before any git changes made
  SSHPubKeyFunc: defaultNoAuthKeyLookup, // Insecure default auth that allows anyone in
  SSHPort:       2222,          // SSH service port
}
```


## Receiver

In Git, The first script to run when handling a push from a client is pre-receive.
It takes a list of references that are being pushed from stdin; if it exits non-zero,
none of them are accepted. [More on hooks](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks).

```go
package main

import (
  "log"
  "os"
  "fmt"

  "github.com/sosedoff/gitkit"
)

// HookInfo contains information about branch, before and after revisions.
// tmpPath is a temporary directory with checked out git tree for the commit.
func receive(hook *gitkit.HookInfo, tmpPath string) error {
  log.Println("Ref:", hook.Ref)
  log.Println("Old revision:", hook.OldRev)
  log.Println("New revision:", hook.NewRev)

  // Check if push is non fast-forward (force)
  force, err := gitkit.IsForcePush(hook)
  if err != nil {
    return err
  }

  // Reject force push
  if force {
    return fmt.Errorf("non fast-forward pushed are not allowed")
  }

  // Getting a commit message is built-in
  message, err := gitkit.ReadCommitMessage(hook.NewRev)
  if err != nil {
    return err
  }

  // Checking on user action
  // Returns one of: branch.push, branch.create, branch.delete, tag.create, tag.delete
  action := hook.Action()

  log.Println("Message:", message)
  return nil
}

func main() {
  receiver := gitkit.Receiver{
    MasterOnly:  false,         // if set to true, only pushes to master branch will be allowed
    TmpDir:      "/tmp/gitkit", // directory for temporary git checkouts
    HandlerFunc: receive,       // your handler function
  }

  // Git hook data is provided via STDIN
  if err := receiver.Handle(os.Stdin); err != nil {
    log.Println("Error:", err)
    os.Exit(1) // terminating with non-zero status will cancel push
  }
}
```

To test if receiver works, you will need to add a sample pre-receive hook to any
git repo. With `go run` its easier to debug but final script should be compiled
and will run very fast.

```bash
#!/bin/bash
cat | go run /path/to/your-receiver.go
```

Modify something in the repo, commit the change and push:

```bash
$ git push
# Counting objects: 3, done.
# Delta compression using up to 8 threads.
# Compressing objects: 100% (3/3), done.
# Writing objects: 100% (3/3), 286 bytes | 0 bytes/s, done.
# Total 3 (delta 2), reused 0 (delta 0)
# -------------------------- out receiver output is here ----------------
# remote: 2016/05/24 17:21:37 Ref: refs/heads/master
# remote: 2016/05/24 17:21:37 Old revision: 5ee8d0891d1e5574e427dc16e0908cb9d28551b9
# remote: 2016/05/24 17:21:37 New revision: e13d6b3a27403029fe674e7b911efd468b035a33
# remote: 2016/05/24 17:21:37 Message: Remove stuff
# To git@localhost:dummy-app.git
#    5ee8d08..e13d6b3  master -> master
```

## Extras

### Remove remote: prefix

If your pre-receive script logs anything to STDOUT, the output might look
like this:

```bash
# Writing objects: 100% (3/3), 286 bytes | 0 bytes/s, done.
# Total 3 (delta 2), reused 0 (delta 0)
remote: Sample script output <---- YOUR SCRIPT
```

There's a simple hack to remove this nasty `remote:` prefix:

```bash
#!/bin/bash
/my/receiver-script | sed -u "s/^/"$'\e[1G\e[K'"/"
```

If you're running on OSX, use `gsed` instead: `brew install gnu-sed`.

Result:

```bash
# Writing objects: 100% (3/3), 286 bytes | 0 bytes/s, done.
# Total 3 (delta 2), reused 0 (delta 0)
Sample script output
```

## Sources

Gitkit contains samples of code ported from the following projects:

- [Flynn](https://github.com/flynn/flynn)
- [Deis](https://github.com/deis/builder)
- [Gitlab Workhorse](https://gitlab.com/gitlab-org/gitlab-workhorse)
- [Gogs](https://github.com/gogits/gogs)

Git docs:

- https://git-scm.com/book/en/v2/Git-Internals-Transfer-Protocols

## License

MIT
