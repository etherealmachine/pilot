package tv

import (
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"time"

	"github.com/etherealmachine/pilot/cec"
	"github.com/etherealmachine/pilot/vlcctrl"
)

type TV interface {
	Playing() string
	Paused() bool
	CECErr() error
	Play(filename string) error
	Pause() error
	Stop() error
	Seek(d time.Duration) error
	Position() time.Duration
	Duration() time.Duration
}

type tv struct {
	cecErr  error
	root    string
	player  *vlcctrl.VLC
}

// New returns a new TV.
func New(root string) (TV, error) {
	player, err := vlcctrl.NewVLC("127.0.0.1", 8081, "raspberry")
	if err != nil {
		return nil, err
	}
	if _, err = player.GetStatus(); err != nil {
		return nil, fmt.Errorf("Error, expected VLC on port 8081, got: %s", err)
	}
	t := &tv{
		root: root,
		player: &player,
	}
	conn, err := cec.Open("", "pilot")
	if err != nil {
		t.cecErr = err
		return t, err
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
		if err := t.Stop(); err != nil {
			log.Println(err)
			t.cecErr = err
		}
	})
	conn.On(cec.FastForward, func() {
		if err := t.Seek(60); err != nil {
			log.Println(err)
			t.cecErr = err
		}
	})
	conn.On(cec.Rewind, func() {
		if err := t.Seek(-60); err != nil {
			log.Println(err)
			t.cecErr = err
		}
	})
	return t, nil
}

func (tv *tv) Playing() string {
	playlist, err := tv.player.Playlist()
	if err != nil {
		log.Println(err)
		return ""
	}
	if len(playlist.Children) < 1 {
		return ""
	}
	if len(playlist.Children[0].Children) < 1 {
		return ""
	}
	return playlist.Children[0].Children[0].Name
}

func (tv *tv) Paused() bool {
	status, err := tv.player.GetStatus()
	if err != nil {
		log.Println(err)
		return false
	}
	return status.State == "paused"
}

func (tv *tv) CECErr() error {
	return tv.cecErr
}

func (tv *tv) Position() time.Duration {
	status, err := tv.player.GetStatus()
	if err != nil {
		log.Println(err)
	}
	return time.Duration(status.Time) * time.Second
}

func (tv *tv) Duration() time.Duration {
	playlist, err := tv.player.Playlist()
	if err != nil {
		log.Println(err)
		return 0
	}
	if len(playlist.Children) < 1 {
		return 0
	}
	if len(playlist.Children[0].Children) < 1 {
		return 0
	}
	return time.Duration(playlist.Children[0].Children[0].Duration) * time.Second
}

func (tv *tv) Play(filename string) error {
	fullpath := filepath.Join(tv.root, filename)
	log.Println("playing", fullpath)
	if err := tv.player.Stop(); err != nil {
		return err
	}
	if err := tv.player.EmptyPlaylist(); err != nil {
		return err
	}
	return tv.player.AddStart(fmt.Sprintf("file://%s", url.PathEscape(fullpath)))
}

func (tv *tv) Pause() error {
	return tv.player.Pause()
}

func (tv *tv) Stop() error {
	if err := tv.player.EmptyPlaylist(); err != nil {
		return err
	}
	return tv.player.Stop()
}

func (tv *tv) Seek(d time.Duration) error {
	return tv.player.Seek(fmt.Sprintf("%d", int64(d / time.Microsecond)))
}
