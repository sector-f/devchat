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
		// Trim leading slash character from "//command" and send "/command" as a normal message
		if len(args[0]) >= 2 {
			if args[0][1] == '/' {
				events <- chatMsgEvent{sender: u.name, msg: strings.Join(args, " ")[1:]} //
				return
			}
		}

		events <- noOpEvent{u} // Trigger a re-render so that the /command doesn't stay on the prompt
	}
}

func clearCMD(_ string, u *user) {
	u.term.Write([]byte("\033[H\033[2J"))
}
