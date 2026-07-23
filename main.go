package main

import (
	"fmt"
	"os"

	"github.com/jim-ww/gowebwrap/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "gowebwrap:", err)
		os.Exit(1)
	}
}
