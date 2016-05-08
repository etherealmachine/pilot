package tv

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/etherealmachine/cec"
)

var (
	player = flag.String("player", "omxplayer", "Video player binary.")
)

type TV struct {
	Playing  string
	Paused   bool
	root     string
	player   *exec.Cmd
	playerIn io.WriteCloser
	cmdout   []string
	cmderr   []string
}

func New(root string) *TV {
	conn, err := cec.Open("", "pilot")
	if err != nil {
		panic(err)
	}
	t := &TV{
		root: root,
	}
	conn.On(cec.Pause, func() {
		if t.Paused {
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
		if t.Paused {
			if err := t.Play(""); err != nil {
				log.Println(err)
			}
		}
	})
	conn.On(cec.Stop, func() {
		if t.player != nil && t.Playing != "" {
			if err := t.Stop(); err != nil {
				log.Println(err)
			}
		}
	})
	return t
}

func (tv *TV) logCmd(cmd *exec.Cmd) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println(err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Println(err)
		return
	}
	go func() {
		s := bufio.NewScanner(stdout)
		for ok := s.Scan(); ok; ok = s.Scan() {
			tv.cmdout = append(tv.cmdout, s.Text())
		}
		if s.Err() != nil {
			tv.cmderr = append(tv.cmderr, s.Err().Error())
		}
	}()
	go func() {
		s := bufio.NewScanner(stderr)
		for ok := s.Scan(); ok; ok = s.Scan() {
			tv.cmdout = append(tv.cmdout, s.Text())
		}
		if s.Err() != nil {
			tv.cmderr = append(tv.cmderr, s.Err().Error())
		}
	}()
}

func (tv *TV) Play(filename string) error {
	if tv.Paused {
		if err := dbusSend("int32:16"); err != nil {
			return err
		}
		tv.Paused = false
		return nil
	} else if tv.Playing != "" {
		log.Println("attempt to play on a running player")
		return nil
	}
	tv.player = exec.Command(*player, filepath.Join(tv.root, filename))
	tv.logCmd(tv.player)
	if in, err := tv.player.StdinPipe(); err != nil {
		return err
	} else {
		tv.playerIn = in
	}
	if err := tv.player.Start(); err != nil {
		return err
	}
	tv.Playing = filename
	tv.Paused = false
	return nil
}

func (tv *TV) Pause() error {
	if tv.player == nil {
		log.Println("attempt to pause a non-running player")
		return nil
	}
	if tv.Paused {
		log.Println("attempt to pause a paused player")
		return nil
	}
	if err := dbusSend("int32:16"); err != nil {
		return err
	}
	tv.Paused = true
	return nil
}

func (tv *TV) Stop() error {
	if tv.player == nil {
		log.Println("attempt to stop a non-running player")
		return nil
	}
	err := dbusSend("int32:15")
	if err != nil {
		log.Println(err)
	}
	err = tv.player.Process.Kill()
	if err != nil {
		log.Println(err)
	}
	err = tv.player.Wait()
	if err != nil {
		log.Println(err)
	}
	tv.player = nil
	tv.Playing = ""
	tv.Paused = false
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
