package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/gliderlabs/ssh"
)

const (
	maxMsgLen = 5120
)

var (
	systemUsername = green.Paint("devbot")
)

type server struct {
	conf    config
	users   []*user
	backlog []backlogMessage
	bans    *banlist

	logger      *log.Logger
	startupTime time.Time

	mu sync.Mutex
}

func newServer(c config) (*server, error) {
	bans, err := banlistFromFile(c.banFilename)
	if err != nil {
		return nil, err
	}

	s := server{
		conf:    c,
		users:   []*user{},
		backlog: []backlogMessage{},
		bans:    bans,

		logger:      log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
		startupTime: time.Now(),

		mu: sync.Mutex{},
	}

	return &s, nil
}

func (s *server) run() error {
	ssh.Handle(func(sess ssh.Session) {
		s.logger.Printf("User %v connecting from %v\n", sess.User(), sess.RemoteAddr())

		u, err := newUser(s, sess)
		if err != nil {
			s.logger.Println(err)
			return
		}

		defer func() { // crash protection
			if i := recover(); i != nil {
				s.broadcast(systemUsername, "Recovered from panic: "+fmt.Sprint(i)+", stack: "+string(debug.Stack()))
			}
		}()

		s.addUser(u)
		s.repl(u)
	})

	s.logger.Printf("Starting server on port %v\n", s.conf.port)
	return ssh.ListenAndServe(
		fmt.Sprintf(":%d", s.conf.port),
		nil,
		ssh.HostKeyFile(s.conf.keyFilename),
		ssh.PublicKeyAuth(
			func(ctx ssh.Context, key ssh.PublicKey) bool {
				return true // allow all keys, this lets us hash pubkeys later
			},
		),
	)
}

func (s *server) broadcast(sender, msg string) {
	if msg == "" {
		return
	}

	rcvTime := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	splitMsg := strings.Split(msg, " ")
	for i := range splitMsg {
		word := splitMsg[i]
		if word == "@everyone" {
			splitMsg[i] = green.Paint("@everyone\a")
		}
	}
	msg = strings.Join(splitMsg, " ")

	for _, u := range s.users {
		u.writeln(sender, msg)
	}

	s.backlog = append(s.backlog, backlogMessage{rcvTime, sender, msg + "\n"})
	if len(s.backlog) > s.conf.scrollback {
		s.backlog = s.backlog[len(s.backlog)-s.conf.scrollback:]
	}
}

func (s *server) addUser(u *user) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users = append(s.users, u)
}

func (s *server) removeUser(u *user) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		index int  = 0
		found bool = false
	)

	for i, user := range s.users {
		if u == user {
			index = i
			found = true
			break
		}
	}

	if !found {
		return errors.New("user does not exist")
	}

	// Replace user that we are removing with the last user in the slice,
	// then remove the last item from the slice
	s.users[index] = s.users[len(s.users)-1]
	s.users = s.users[:len(s.users)-1]

	return nil
}

type backlogMessage struct {
	timestamp  time.Time
	senderName string
	text       string
}
