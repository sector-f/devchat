package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/gliderlabs/ssh"
)

const (
	maxMsgLen = 5120
)

var (
	systemUsername = green.Paint("SYSTEM")
	timeFormat     = "15:04:05"
)

type server struct {
	conf    config
	users   []*user
	backlog []event
	bans    *banlist

	logger      *log.Logger
	startupTime time.Time

	mu     sync.Mutex
	events chan event
}

func newServer(c config) (*server, error) {
	bans, err := banlistFromFile(c.banFilename)
	if err != nil {
		return nil, err
	}

	s := server{
		conf:    c,
		users:   []*user{},
		backlog: []event{},
		bans:    bans,

		logger:      log.New(os.Stdout, "", log.Ldate|log.Ltime),
		startupTime: time.Now(),

		mu:     sync.Mutex{},
		events: make(chan event),
	}

	return &s, nil
}

func (s *server) run() func() {
	sshServer := ssh.Server{
		Addr: fmt.Sprintf(":%d", s.conf.port),
		Handler: func(sess ssh.Session) {
			s.logger.Printf("User %s [%s] connecting from %v\n", sess.User(), shasum(string(sess.PublicKey().Marshal()))[:8], formatAddr(sess.RemoteAddr()))

			u, err := newUser(s, sess)
			if err != nil {
				s.logger.Println(err)
				return
			}

			defer func() { // crash protection
				if i := recover(); i != nil {
					s.events <- systemMsgEvent{msg: "Recovered from panic: " + fmt.Sprint(i) + ", stack: " + string(debug.Stack())}
				}
			}()

			s.repl(u)
		},
	}

	sshServer.SetOption(ssh.HostKeyFile(s.conf.keyFilename))
	sshServer.SetOption(
		ssh.PublicKeyAuth(
			func(ctx ssh.Context, key ssh.PublicKey) bool {
				return true // allow all keys, this lets us hash pubkeys later
			},
		),
	)

	s.logger.Printf("Starting server on port %v\n", s.conf.port)

	go func() {
		sshServer.ListenAndServe()
	}()

	go func() {
		for rcvd := range s.events {
			rcvdAt := time.Now()

			s.logger.Println(stripansi.Strip(rcvd.Sender() + ": " + rcvd.Message()))

			s.backlog = append(s.backlog, rcvd)
			if len(s.backlog) > s.conf.scrollback {
				s.backlog = s.backlog[len(s.backlog)-s.conf.scrollback:]
			}

			switch event := rcvd.(type) {
			case joinEvent:
				s.addUser(event.user)
				for _, user := range s.users {
					user.events <- joinEvent{event.user, rcvdAt}
				}
			case partEvent:
				s.removeUserQuietly(event.user)
				for _, user := range s.users {
					user.events <- partEvent{event.user, event.reason, rcvdAt}
				}
			case chatMsgEvent:
				for _, user := range s.users {
					user.events <- chatMsgEvent{event.sender, event.msg, rcvdAt}
				}
			case systemMsgEvent:
				for _, user := range s.users {
					user.events <- systemMsgEvent{event.msg, rcvdAt}
				}
			case shutdownEvent:
				sshServer.Close()
				return
			default:
				s.logger.Println("Received invalid type on message channel")
			}

		}
	}()

	return func() {
		s.events <- shutdownEvent{}

		err := s.bans.save()
		if err != nil {
			s.logger.Printf("Error saving bans: %v", err)
		}
	}
}

func (s *server) addUser(u *user) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users = append(s.users, u)
}

func (s *server) removeUserQuietly(u *user) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u.session.Close()

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
		return
	}

	// Replace user that we are removing with the last user in the slice,
	// then remove the last item from the slice
	s.users[index] = s.users[len(s.users)-1]
	s.users = s.users[:len(s.users)-1]

	return
}
