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

	shutdownServer := s.run()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	<-c
	shutdownServer()
}
