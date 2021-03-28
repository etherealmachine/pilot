package cec

// #include <libcec/cecc.h>
import "C"

import (
	"log"
	"regexp"
	"unsafe"
)

type EventType int

const (
	Unknown = EventType(iota)
	Pause
	Play
	Stop
	FastForward
	Rewind
)

type regexEvent struct {
	re    *regexp.Regexp
	event EventType
}

var events = []*regexEvent{
	{
		regexp.MustCompile(`key pressed: pause \([0-9]+, [0-9]+\)`),
		Pause,
	},
	{
		regexp.MustCompile(`key pressed: play \([0-9]+, [0-9]+\)`),
		Play,
	},
	{
		regexp.MustCompile(`key pressed: stop \([0-9]+\) current\(ff\) duration\(0\)`),
		Stop,
	},
	{
		regexp.MustCompile(`key pressed: Fast forward \([0-9]+, [0-9]+\)`),
		FastForward,
	},
	{
		regexp.MustCompile(`key pressed: rewind \([0-9]+, [0-9]+\)`),
		Rewind,
	},
}

//export logMessageCallback
func logMessageCallback(c unsafe.Pointer, msg *C.cec_log_message) {
	s := C.GoString(msg.message)
	for _, conn := range connections {
		conn.recvMsg(s)
	}
	log.Println(s)
}

func (c *Connection) On(e EventType, callback func()) {
	c.events[e] = callback
}

func (c *Connection) recvMsg(msg string) {
	e := Unknown
	for _, evt := range events {
		if evt.re.MatchString(msg) {
			e = evt.event
		}
	}
	if cb := c.events[e]; cb != nil {
		cb()
	}
}
