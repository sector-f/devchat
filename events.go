package main

import "time"

type event interface {
	Sender() string
	Message() string
	ReceivedAt() time.Time
}

type joinEvent struct {
	user   *user
	rcvdAt time.Time
}

func (e joinEvent) Sender() string {
	return systemUsername
}

func (e joinEvent) Message() string {
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

func (e partEvent) Sender() string {
	return systemUsername
}

func (e partEvent) Message() string {
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

func (e chatMsgEvent) Sender() string {
	return e.sender
}

func (e chatMsgEvent) Message() string {
	return e.msg
}

func (e chatMsgEvent) ReceivedAt() time.Time {
	return e.rcvdAt
}

type systemMsgEvent struct {
	msg    string
	rcvdAt time.Time
}

func (e systemMsgEvent) Sender() string {
	return systemUsername
}

func (e systemMsgEvent) Message() string {
	return e.msg
}

func (e systemMsgEvent) ReceivedAt() time.Time {
	return e.rcvdAt
}

type shutdownEvent struct {
	rcvdAt time.Time
}

func (e shutdownEvent) Sender() string {
	return systemUsername
}

func (e shutdownEvent) Message() string {
	return "Server is shutting down"
}

func (e shutdownEvent) ReceivedAt() time.Time {
	return e.rcvdAt
}

// Does nothing; used to trigger a re-render for user
type noOpEvent struct {
	user *user
}

func (e noOpEvent) Sender() string        { return "" }
func (e noOpEvent) Message() string       { return "" }
func (e noOpEvent) ReceivedAt() time.Time { return time.Time{} }
