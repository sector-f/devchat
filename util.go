package main

import (
	"net"
	"strings"
	"time"

	"github.com/acarl005/stripansi"
)

func formatAddr(n net.Addr) string {
	networkStr := n.String()
	host, _, _ := net.SplitHostPort(networkStr)
	return host
}

func printPrettyDuration(d time.Duration) string {
	s := strings.TrimSpace(strings.TrimSuffix(d.Round(time.Minute).String(), "0s"))
	if s == "" { // we cut off the seconds so if there's nothing in the string it means it was made of only seconds.
		s = "< 1m"
	}
	return s
}

func lenString(a string) int {
	return len([]rune(stripansi.Strip(a)))
}
