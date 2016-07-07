package main

import (
	"github.com/BTBurke/gitkit"
	"log"
	"os"
)

func main() {
	log.New(os.Stdout, "", 0)
	server := gitkit.New()
	server.Run()
}
