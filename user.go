package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"net"
	"strings"
	"time"
	"unicode"

	"github.com/acarl005/stripansi"
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

	bell    bool
	colorBG string
	id      string
	addr    string

	lastTimestamp time.Time
	joinTime      time.Time
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
		name:          "",
		pronouns:      []string{"unset"},
		session:       sess,
		term:          term,
		bell:          true,
		colorBG:       "bg-off", // the FG will be set randomly
		id:            shasum(toHash),
		addr:          host,
		win:           w,
		lastTimestamp: now,
		joinTime:      now,
	}

	clearCMD("", u) // always clear the screen on connect

	err := s.setUsername(u, sess.User())
	if err != nil {
		s.writeln(u, systemUsername, "Error setting name:"+err.Error())
		sess.Close()
		return nil, err
	}

	go func() {
		for u.win = range winChan {
		}
	}()

	if s.bans.contains(u.id) {
		s.logger.Printf("Rejected %s [%s]\n", u.name, host)
		s.writeln(u, systemUsername, "You are banned")
		s.removeUserQuietly(u)
		return nil, errors.New("user is banned")
	}

	// TOOD: maybe replace with leaky bucket?
	/*
		idsInMinToTimes[u.id]++
		time.AfterFunc(60*time.Second, func() {
			idsInMinToTimes[u.id]--
		})
		if idsInMinToTimes[u.id] > 6 {
			bans = append(bans, ban{u.addr, u.id})
			mainRoom.broadcast(devbot, "`"+sess.User()+"` has been banned automatically. ID: "+u.id)
			return nil
		}
	*/

	if len(s.backlog) > 0 {
		lastStamp := s.backlog[0].timestamp
		u.rWriteln(printPrettyDuration(u.joinTime.Sub(lastStamp)) + " earlier")
		for _, msg := range s.backlog {
			if msg.timestamp.Sub(lastStamp) > time.Minute {
				lastStamp = msg.timestamp
				u.rWriteln(printPrettyDuration(u.joinTime.Sub(lastStamp)) + " earlier")
			}
			s.writeln(u, msg.senderName, msg.text)
		}
	}

	/*
		switch len(s.mainRoom.users) - 1 {
		case 0:
			u.writeln("", blue.Paint("Welcome to the chat. There are no more users"))
		case 1:
			u.writeln("", yellow.Paint("Welcome to the chat. There is one more user"))
		default:
			u.writeln("", green.Paint("Welcome to the chat. There are", strconv.Itoa(len(s.mainRoom.users)-1), "more users"))
		}
		s.mainRoom.broadcast(systemUsername, u.name+" has joined the chat")
	*/

	return u, nil
}

// TODO: figure out which file this should be in
func (s *server) repl(u *user) {
	for {
		line, err := u.term.ReadLine()
		switch err {
		case io.EOF:
			s.removeUser(u, u.name+" has left the chat")
			return
		case nil:
			// Do nothing
		default:
			s.removeUser(u, u.name+" has left the chat due to an error: "+err.Error())
			return
		}
		line += "\n"

		// Limit message length as early as possible
		// TODO: see if splitting into multiple messages is possible (and a good idea)
		if len(line) > maxMsgLen {
			line = line[0:maxMsgLen]
		}

		line = strings.TrimRightFunc(line, unicode.IsSpace)
		u.term.SetPrompt(u.name + ": ")
		u.term.Write([]byte(strings.Repeat("\033[A\033[2K", int(math.Ceil(float64(lenString(u.name+line)+2)/(float64(u.win.Width))))))) // basically, ceil(length of line divided by term width)

		if line == "" {
			continue
		}

		// TODO: command handling goes here
		s.broadcast(u.name, line)
	}
}

func (s *server) writeln(u *user, sender, msg string) {
	msg = sender + ": " + msg
	if !u.bell {
		msg = strings.ReplaceAll(msg, "\a", "")
	}

	_, err := u.term.Write([]byte(msg + "\n"))
	if err != nil {
		s.removeUser(u, u.name+"has left the chat because of an error writing to their terminal: "+err.Error())
	}
}

func (u *user) rWriteln(msg string) {}

func (s *server) setUsername(u *user, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	taken := false
	for _, user := range s.users {
		if user.name == u.name {
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

func printPrettyDuration(d time.Duration) string {
	s := strings.TrimSpace(strings.TrimSuffix(d.Round(time.Minute).String(), "0s"))
	if s == "" { // we cut off the seconds so if there's nothing in the string it means it was made of only seconds.
		s = "< 1m"
	}
	return s
}

func lenString(a string) int {
	return len([]rune(stripansi.Strip(a)))
}
