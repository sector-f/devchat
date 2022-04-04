package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/term"
)

// Map of SSH key hash to user
type users map[string]*user

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
	room          *room
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
		room:          s.mainRoom}

	go func() {
		for u.win = range winChan {
		}
	}()

	s.logger.Printf("Connected %s [%s]\n", u.name, u.id)

	if s.bans.contains(u.id) {
		s.logger.Printf("Rejected %s [%s]\n", u.name, host)
		u.writeln(systemUsername, "**You are banned**. If you feel this was a mistake, please reach out at github.com/quackduck/devzat/issues or email igoel.mail@gmail.com. Please include the following information: [ID "+u.id+"]")
		u.closeQuietly()
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

	clearCMD("", u) // always clear the screen on connect

	if len(s.backlog) > 0 {
		lastStamp := s.backlog[0].timestamp
		u.rWriteln(printPrettyDuration(u.joinTime.Sub(lastStamp)) + " earlier")
		for _, msg := range s.backlog {
			if msg.timestamp.Sub(lastStamp) > time.Minute {
				lastStamp = msg.timestamp
				u.rWriteln(printPrettyDuration(u.joinTime.Sub(lastStamp)) + " earlier")
			}
			u.writeln(msg.senderName, msg.text)
		}
	}

	if err := u.pickUsernameQuietly(sess.User()); err != nil { // user exited or had some error
		return nil, err
	}

	/*

		// TODO: this should probably be a method of room
		s.mainRoom.usersMutex.Lock()
		s.mainRoom.users = append(mainRoom.users, u)
		s.mainRoom.usersMutex.Unlock()

		u.term.SetBracketedPasteMode(true) // experimental paste bracketing support
		term.AutoCompleteCallback = func(line string, pos int, key rune) (string, int, bool) {
			return autocompleteCallback(u, line, pos, key)
		}

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

func (u *user) repl() {}

func (u *user) writeln(sender, msg string) {}

func (u *user) rWriteln(msg string) {}

func (u *user) closeQuietly() {}

func (u *user) pickUsernameQuietly(name string) error { return errors.New("Unimplemented") }

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
