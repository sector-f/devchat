package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config := defaultConfig()

	s, err := newServer(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	shutdownServer := s.run()

	<-c
	shutdownServer()
}
