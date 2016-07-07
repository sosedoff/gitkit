package main

import (
	"crypto/subtle"
	"fmt"
	"github.com/BTBurke/gitkit"
	"log"
	"os"
)

func main() {
	fmt.Println("Running with basic auth.  Try pushing a commit with http://hello:reallylongpassword@127.0.0.1:8080/test.git")
	log.New(os.Stdout, "", 0)

	myAuthFunc := func(c gitkit.Credential, r *gitkit.Request) (bool, error) {
		myPasswords := map[string][]byte{
			"hello": []byte("reallylongpassword"),
		}
		return subtle.ConstantTimeCompare(myPasswords[c.Username], []byte(c.Password)) == 1, nil
	}

	auth := gitkit.UseAuthFunc(myAuthFunc)
	server := gitkit.New(auth)
	server.Run()
}
