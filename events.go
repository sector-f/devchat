package main

type joinEvent struct {
	user *user
}

type partEvent struct {
	username string
	reason   string
}

type chatMsgEvent struct {
	sender string
	msg    string
}

type systemMsgEvent struct {
	msg string
}

type shutdownEvent struct{}
