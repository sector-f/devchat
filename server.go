package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
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

	mu         sync.Mutex
	events     chan event
	isShutdown chan struct{} // Used to block until server is fully shut down
}

func newServer(c config) (*server, error) {
	bans, err := banlistFromFile(c.BanFilename)
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

		mu:         sync.Mutex{},
		events:     make(chan event),
		isShutdown: make(chan struct{}),
	}

	return &s, nil
}

func (s *server) run() func() {
	sshServer := ssh.Server{
		Addr: fmt.Sprintf(":%d", s.conf.Port),
		Handler: func(sess ssh.Session) {
			s.logger.Printf("User %s [%s] connecting from %v\n", sess.User(), shasum(string(sess.PublicKey().Marshal()))[:8], formatAddr(sess.RemoteAddr()))

			u, err := newUser(s, sess)
			if err != nil {
				s.logger.Println(err)
				return
			}

			defer func() { // crash protection
				if i := recover(); i != nil {
					s.events <- &systemMsgEvent{msg: "Recovered from panic: " + fmt.Sprint(i) + ", stack: " + string(debug.Stack())}
				}
			}()

			repl(u, s.events)
		},
	}

	sshServer.SetOption(ssh.HostKeyFile(s.conf.KeyFilename))
	sshServer.SetOption(
		ssh.PublicKeyAuth(
			func(ctx ssh.Context, key ssh.PublicKey) bool {
				return true // allow all keys, this lets us hash pubkeys later
			},
		),
	)

	s.logger.Printf("Starting server on port %v\n", s.conf.Port)

	go func() {
		sshServer.ListenAndServe()
	}()

	go func() {
		defer func() {
			s.isShutdown <- struct{}{}
		}()

		for rcvd := range s.events {
			rcvdAt := time.Now()
			rcvd.SetReceivedAt(rcvdAt)

			if rcvd.ShouldLog() {
				s.logger.Println(stripansi.Strip(rcvd.Sender() + ": " + rcvd.Message()))

				s.backlog = append(s.backlog, rcvd)
				if len(s.backlog) > s.conf.Scrollback {
					s.backlog = s.backlog[len(s.backlog)-s.conf.Scrollback:]
				}
			}

			switch event := rcvd.(type) {
			case *joinEvent:
				s.users = append(s.users, event.user)

				for _, user := range s.users {
					user.events <- &joinEvent{event.user, rcvdAt}
				}
			case *partEvent:
				s.removeUserQuietly(event.user)
				for _, user := range s.users {
					user.events <- &partEvent{event.user, event.reason, rcvdAt}
				}
			case *chatMsgEvent:
				for _, user := range s.users {
					user.events <- &chatMsgEvent{event.sender, event.msg, rcvdAt}
				}
			case *whisperMsgEvent:
				if event.sender.name == event.receiver {
					event.sender.events <- &systemWhisperMsgEvent{msg: "whisper: you cannot message yourself", rcvdAt: rcvdAt}
					continue
				}

				var rcvUser *user
				rcvrExists := false
				for _, user := range s.users {
					if user.name == event.receiver {
						rcvUser = user
						rcvrExists = true
						break
					}
				}

				if !rcvrExists {
					event.sender.events <- &systemWhisperMsgEvent{msg: "whisper: user does not exist", rcvdAt: rcvdAt}
					continue
				}

				event.sender.events <- &whisperMsgEvent{sender: event.sender, receiver: event.receiver, msg: event.msg, rcvdAt: rcvdAt}
				rcvUser.events <- &whisperMsgEvent{sender: event.sender, receiver: event.receiver, msg: event.msg, rcvdAt: rcvdAt}
			case *systemMsgEvent:
				for _, user := range s.users {
					user.events <- &systemMsgEvent{event.msg, rcvdAt}
				}
			case *systemWhisperMsgEvent:
				event.receiver.events <- &systemWhisperMsgEvent{msg: event.msg, rcvdAt: rcvdAt}
			case *shutdownEvent:
				c := make(chan os.Signal, 2)
				signal.Notify(c, os.Interrupt, syscall.SIGTERM)
				go func() {
					<-c
					s.logger.Println("Caught signal again, shutting down immediately")
					os.Exit(1)
				}()

				for _, user := range s.users {
					user.events <- &shutdownEvent{rcvdAt: rcvdAt}
				}

				ctx, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)
				sshServer.Shutdown(ctx)

				<-ctx.Done()
				sshServer.Close()

				err := s.bans.save()
				if err != nil {
					s.logger.Printf("Error saving bans: %v", err)
				}

				return
			case noOpEvent:
				event.user.render()
			default:
				s.logger.Println("Received invalid type on message channel")
			}

		}
	}()

	return func() {
		s.events <- &shutdownEvent{}
		<-s.isShutdown
	}
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
