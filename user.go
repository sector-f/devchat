package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/term"
)

type user struct {
	name     string
	pronouns []string

	session ssh.Session
	term    *term.Terminal
	win     ssh.Window
	backlog []event
	events  chan event

	bell    bool
	colorBG string
	id      string
	addr    string

	lastTimestamp time.Time
	joinTime      time.Time

	mu sync.Mutex
}

func (u *user) render() {
	u.mu.Lock()
	defer u.mu.Unlock()

	clearCMD("", u)

	// TODO: this probably doesn't handle bells correctly
	if len(u.backlog) > 0 {
		for _, msg := range u.backlog {
			u.writeln(msg.Sender(), msg.Message(), msg.ReceivedAt().Format(timeFormat))
		}

		windowWidth := u.win.Width
		if windowWidth > 0 {
			u.term.Write([]byte(darkGreen.Paint(strings.Repeat("━", windowWidth)) + "\n"))
		}
	}

	u.term.SetPrompt(u.name + ": ")
}

func newUser(s *server, sess ssh.Session) (*user, error) {
	term := terminal.NewTerminal(sess, "> ")
	_ = term.SetSize(10000, 10000) // disable any formatting done by term
	pty, winChan, _ := sess.Pty()
	w := pty.Window
	host, _, _ := net.SplitHostPort(sess.RemoteAddr().String()) // definitely should not give an err

	pubkey := sess.PublicKey()
	if pubkey == nil { // TODO: figure out if this can actually occur
		return nil, errors.New("public key was nil")
	}

	toHash := string(pubkey.Marshal())

	now := time.Now()
	u := &user{
		name:     "",
		pronouns: []string{"unset"},

		session: sess,
		term:    term,
		win:     w,
		backlog: s.backlog,
		events:  make(chan event),

		bell:    true,
		colorBG: "bg-off", // the FG will be set randomly
		id:      shasum(toHash),
		addr:    host,

		lastTimestamp: now,
		joinTime:      now,
	}

	clearCMD("", u) // always clear the screen on connect

	err := s.setUsername(u, sess.User())
	if err != nil {
		u.writeln(systemUsername, "Error setting name: "+err.Error(), "")
		sess.Close()
		return nil, err
	}

	if s.bans.contains(u.id) {
		s.logger.Printf("Rejected %s [%s]\n", u.name, host)
		u.writeln(systemUsername, "You are banned", "")
		s.removeUserQuietly(u)
		return nil, errors.New("user is banned")
	}

	go func() {
		for u.win = range winChan {
			u.render()
		}
	}()

	s.events <- joinEvent{user: u}
	return u, nil
}

// TODO: figure out which file this should be in
func (s *server) repl(u *user) {
	go func() {
		for event := range u.events {
			u.backlog = append(u.backlog, event)
			u.render()
		}
	}()

	for {
		line, err := u.term.ReadLine()

		switch err {
		case io.EOF:
			s.events <- partEvent{user: u, reason: "quit"}
			return
		case nil:
			// Do nothing
		default:
			s.events <- partEvent{user: u, reason: "Error: " + err.Error()}
			return
		}

		// Limit message length as early as possible
		// TODO: see if splitting into multiple messages is possible (and a good idea)
		if len(line) > maxMsgLen {
			line = line[0:maxMsgLen]
		}

		line = strings.TrimRightFunc(line, unicode.IsSpace)

		if line == "" {
			continue
		}

		// TODO: command handling goes here
		s.events <- chatMsgEvent{sender: u.name, msg: line}
	}
}

func (u *user) writeln(sender, msg string, right string) {
	msg = sender + ": " + msg
	if !u.bell {
		msg = strings.ReplaceAll(msg, "\a", "")
	}

	u.term.Write([]byte(msg))
	if right != "" {
		windowWidth := u.win.Width
		msgLen := lenString(msg + right)

		if windowWidth-msgLen > 0 {
			u.term.Write([]byte(strings.Repeat(" ", windowWidth-msgLen) + right + "\n"))
		} else {
			u.term.Write([]byte(right + "\n"))
		}
	} else {
		u.term.Write([]byte("\n"))
	}
}

func (s *server) setUsername(u *user, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	taken := false
	for _, user := range s.users {
		if name == user.name {
			s.logger.Println("It matches")
			taken = true
			break
		}
	}

	if taken {
		return errors.New("name is already taken")
	}

	u.name = name

	return nil
}

// TODO: add color to this eventually?
func (u *user) formatPronouns() string {
	return strings.Join(u.pronouns, "/")
}

func shasum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
