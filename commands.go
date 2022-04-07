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

	switch args[0] {
	case "/whisper":
		switch len(args) {
		case 1:
			events <- systemWhisperMsgEvent{receiver: u, msg: "whisper: no user specified"}
		case 2:
			events <- systemWhisperMsgEvent{receiver: u, msg: "whisper: no message specified"}
		default:
			events <- whisperMsgEvent{sender: u, receiver: args[1], msg: strings.Join(args[2:], " ")}
		}
	default:
		events <- noOpEvent{u}
	}
}

func clearCMD(_ string, u *user) {
	u.term.Write([]byte("\033[H\033[2J"))
}
