package main

func clearCMD(_ string, u *user) {
	u.term.Write([]byte("\033[H\033[2J"))
}
