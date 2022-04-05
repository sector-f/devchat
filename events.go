package main

import "time"

type event interface {
	String() string
	ReceivedAt() time.Time
}

type joinEvent struct {
	user   *user
	rcvdAt time.Time
}

func (e joinEvent) String() string {
	return e.user.name + " has joined"
}

func (e joinEvent) ReceivedAt() time.Time {
	return e.rcvdAt
}

type partEvent struct {
	user   *user
	reason string
	rcvdAt time.Time
}

func (e partEvent) String() string {
	msg := e.user.name + " has left"
	if e.reason != "" {
		msg += " (" + e.reason + ")"
	}

	return msg
}

func (e partEvent) ReceivedAt() time.Time {
	return e.rcvdAt
}

type chatMsgEvent struct {
	sender string
	msg    string
	rcvdAt time.Time
}

func (e chatMsgEvent) String() string {
	return e.sender + ": " + e.msg
}

func (e chatMsgEvent) ReceivedAt() time.Time {
	return e.rcvdAt
}

type systemMsgEvent struct {
	msg    string
	rcvdAt time.Time
}

func (e systemMsgEvent) String() string {
	return e.msg
}

func (e systemMsgEvent) ReceivedAt() time.Time {
	return e.rcvdAt
}

type shutdownEvent struct {
	rcvdAt time.Time
}

func (e shutdownEvent) String() string {
	return "System is shutting down"
}

func (e shutdownEvent) ReceivedAt() time.Time {
	return e.rcvdAt
}
