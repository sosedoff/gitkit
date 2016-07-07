package main

import (
	"fmt"
	"github.com/BTBurke/gitkit"
	"log"
	"os"
)

func main() {
	fmt.Println("This example shows a basic dual HTTP and SSH server running on ports 8080 and 2222 respectively.\n\n**Warning** Don't use this model in production as nothing is secured.  You should look at the other examples for how to configure authorization functions, TLS, and other security measures.\n")
	log.New(os.Stdout, "", 0)
	server := gitkit.New()
	server.Run()
}
