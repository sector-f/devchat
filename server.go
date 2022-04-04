package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
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
	conf     config
	mainRoom *room
	rooms    map[string]*room
	backlog  []backlogMessage
	bans     *banlist

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
		conf:     c,
		mainRoom: &room{name: "#main"},
		rooms:    make(map[string]*room),
		backlog:  []backlogMessage{},
		bans:     bans,

		logger:      log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
		startupTime: time.Now(),

		mu: sync.Mutex{},
	}

	return &s, nil
}

func (s *server) run() error {
	// TODO: see if we can create a concrete instance here rather than relying on
	// package-scoped vars, like ssh.DefaultHandler here
	ssh.Handle(func(sess ssh.Session) {
		u, err := newUser(s, sess)
		if err != nil {
			s.logger.Println(err)
			sess.Close()
			return
		}
		defer func() { // crash protection
			if i := recover(); i != nil {
				s.mainRoom.broadcast(systemUsername, "Slap the developers in the face for me, the server almost crashed, also tell them this: "+fmt.Sprint(i)+", stack: "+string(debug.Stack()))
			}
		}()
		u.repl()
	})

	return ssh.ListenAndServe(
		fmt.Sprintf(":%d", s.conf.port),
		nil,
		ssh.HostKeyFile(os.Getenv("HOME")+"/.ssh/id_rsa"),
		ssh.PublicKeyAuth(
			func(ctx ssh.Context, key ssh.PublicKey) bool {
				return true // allow all keys, this lets us hash pubkeys later
			},
		),
	)
}

type backlogMessage struct {
	timestamp  time.Time
	senderName string
	text       string
}
