package main

import (
	"os"

	"cperfc"
	"cperfc/config"
)

func init() {
}

func main() {
	config.ParseCommandLine()
	os.Exit(cperfc.Run())
}
