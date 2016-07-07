package main

import (
	"github.com/BTBurke/gitkit"
	//kitlog "github.com/go-kit/kit/log"
	"log"
	"os"
)

func main() {
	//logger := kitlog.NewJSONLogger(os.Stdout)
	//log.SetOutput(kitlog.NewStdlibAdapter(logger))
	log.New(os.Stdout, "", 0)
	server := gitkit.New()
	server.Run()
}
