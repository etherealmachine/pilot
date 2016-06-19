package tv

import (
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/etherealmachine/cec"
	"github.com/etherealmachine/omxplayer"
)

type TV interface {
	Playing() string
	Paused() bool
	CECErr() error
	Play(filename string) error
	Pause() error
	Stop() error
	Seek(seconds int) error
}

type tv struct {
	playing string
	cecErr  error
	root    string
	player  *omxplayer.Player
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
		if err := t.Pause(); err != nil {
			log.Println(err)
			t.cecErr = err
		}
	})
	conn.On(cec.Play, func() {
		if err := t.Pause(); err != nil {
			log.Println(err)
			t.cecErr = err
		}
	})
	conn.On(cec.Stop, func() {
		if t.player != nil && t.playing != "" {
			if err := t.Stop(); err != nil {
				log.Println(err)
				t.cecErr = err
			}
		}
	})
	conn.On(cec.FastForward, func() {
		if t.player != nil && t.playing != "" {
			if err := t.Seek(60); err != nil {
				log.Println(err)
				t.cecErr = err
			}
		}
	})
	conn.On(cec.Rewind, func() {
		if t.player != nil && t.playing != "" {
			if err := t.Seek(-60); err != nil {
				log.Println(err)
				t.cecErr = err
			}
		}
	})
	return t
}

func (tv *tv) Playing() string {
	return tv.playing
}

func (tv *tv) Paused() bool {
	if tv.player == nil {
		return false
	}
	status, err := tv.player.PlaybackStatus()
	if err != nil {
		log.Println(err)
		return false
	}
	return status == "Paused"
}

func (tv *tv) CECErr() error {
	return tv.cecErr
}

func (tv *tv) Play(filename string) error {
	if tv.player != nil && tv.player.IsRunning() {
		if err := tv.player.Quit(); err != nil {
			return err
		}
	}
	omxplayer.SetUser("root", "/root")
	var err error
	tv.player, err = omxplayer.New(filepath.Join(tv.root, filename))
	if err != nil {
		return err
	}
	tv.player.WaitForReady()
	if err := tv.player.PlayPause(); err != nil {
		return err
	}
	tv.playing = filename
	return nil
}

func (tv *tv) Pause() error {
	if tv.player == nil {
		return nil
	}
	if err := tv.player.Pause(); err != nil {
		return err
	}
	return nil
}

func (tv *tv) Stop() error {
	if tv.player == nil {
		return nil
	}
	if tv.player.IsRunning() {
		if err := tv.player.Quit(); err != nil {
			return err
		}
	}
	if !tv.player.IsRunning() {
		tv.playing = ""
		tv.player = nil
		exec.Command("killall", "dbus-daemon").Run()
	}
	return nil
}

func (tv *tv) Seek(seconds int) error {
	if tv.player == nil {
		return nil
	}
	_, err := tv.player.Seek(int64(
		time.Duration(seconds) *
			(time.Second / time.Microsecond)))
	return err
}
