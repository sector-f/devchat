package main

import (
	"fmt"
	"os"
)

func main() {
	config := defaultConfig()

	s, err := newServer(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = s.run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
