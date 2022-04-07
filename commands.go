package main

import (
	"strings"
)

func (u *user) parseCommand(input string, events chan event) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	args := strings.Split(input, " ")

	if args[0][0] != '/' {
		events <- chatMsgEvent{sender: u.name, msg: input}
		return
	}

	events <- noOpEvent{u}
}

func clearCMD(_ string, u *user) {
	u.term.Write([]byte("\033[H\033[2J"))
}
