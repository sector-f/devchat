package main

type joinEvent struct {
	username string
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
