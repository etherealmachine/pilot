package tv

import (
	"log"
	"path/filepath"

	"github.com/etherealmachine/cec"
	"github.com/jleight/omxplayer"
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
	return status == "paused"
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
	var err error
	tv.player, err = omxplayer.New(filepath.Join(tv.root, filename))
	if err != nil {
		return err
	}
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
	return nil
}
