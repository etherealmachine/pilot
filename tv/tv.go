package tv

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/etherealmachine/cec"
)

type TV interface {
	Playing() string
	Paused() bool
	CECErr() error
	Play(filename string) error
	Pause() error
	Stop() error
}

type tv struct {
	playing string
	paused  bool
	cecErr  error
	root    string
	player  *exec.Cmd
}

// New returns a new TV.
func New(root string) TV {
	t := &tv{
		root: root,
	}
	conn, err := cec.Open("", "pilot")
	if err != nil {
		t.cecErr = err
		return t
	}
	conn.On(cec.Pause, func() {
		if t.paused {
			if err := t.Play(""); err != nil {
				log.Println(err)
			}
		} else {
			if err := t.Pause(); err != nil {
				log.Println(err)
			}
		}
	})
	conn.On(cec.Play, func() {
		if t.paused {
			if err := t.Play(""); err != nil {
				log.Println(err)
			}
		}
	})
	conn.On(cec.Stop, func() {
		if t.player != nil && t.playing != "" {
			if err := t.Stop(); err != nil {
				log.Println(err)
			}
		}
	})
	return t
}

func (tv *tv) Playing() string {
	return tv.playing
}

func (tv *tv) Paused() bool {
	return tv.paused
}

func (tv *tv) CECErr() error {
	return tv.cecErr
}

func logPipe(prefix string, r io.ReadCloser) {
	log.Printf("logging %s", prefix)
	buf := bufio.NewReader(r)
	for {
		l, err := buf.ReadString('\n')
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("%s: %s", prefix, l)
	}
}

func (tv *tv) Play(filename string) error {
	if tv.paused {
		if err := dbusSend("int32:16"); err != nil {
			return err
		}
		tv.paused = false
		return nil
	} else if tv.player != nil {
		if err := tv.Stop(); err != nil {
			return err
		}
	}
	tv.player = exec.Command("/usr/bin/omxplayer", filepath.Join(tv.root, filename))
	stderr, err := tv.player.StderrPipe()
	if err != nil {
		log.Println(err)
		return err
	}
	stdout, err := tv.player.StdoutPipe()
	if err != nil {
		log.Println(err)
		return err
	}
	err = tv.player.Start()
	if err != nil {
		log.Println(err)
		return err
	}
	go logPipe("omxplayer stderr", stderr)
	go logPipe("omxplayer stdout", stdout)
	tv.playing = filename
	tv.paused = false
	return nil
}

func (tv *tv) Pause() error {
	if tv.player == nil {
		log.Println("attempt to pause a non-running player")
		return nil
	}
	if tv.paused {
		log.Println("attempt to pause a paused player")
		return nil
	}
	if err := dbusSend("int32:16"); err != nil {
		return err
	}
	tv.paused = true
	return nil
}

func (tv *tv) Stop() error {
	if tv.player == nil {
		tv.player = nil
		tv.playing = ""
		tv.paused = false
		log.Println("attempt to stop a non-running player")
		return nil
	}
	if err := dbusSend("int32:15"); err != nil {
		return err
	}
	if err := tv.player.Wait(); err != nil {
		return err
	}
	tv.player = nil
	tv.playing = ""
	tv.paused = false
	return nil
}

func dbusSend(action string) error {
	addr, err := ioutil.ReadFile("/tmp/omxplayerdbus.root")
	if err != nil {
		return err
	}
	pid, err := ioutil.ReadFile("/tmp/omxplayerdbus.root.pid")
	if err != nil {
		return err
	}
	cmd := exec.Command(
		"dbus-send",
		"--print-reply=literal",
		"--session",
		"--dest=org.mpris.MediaPlayer2.omxplayer",
		"/org/mpris/MediaPlayer2",
		"org.mpris.MediaPlayer2.Player.Action",
		action)
	cmd.Env = []string{
		"DBUS_SESSION_BUS_ADDRESS=" + strings.TrimSpace(string(addr)),
		"DBUS_SESSION_BUS_PID=" + strings.TrimSpace(string(pid)),
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}
