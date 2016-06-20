package tv

import (
	"math/rand"
	"path/filepath"
	"time"
)

type mockTV struct {
	root     string
	playing  string
	paused   bool
	ticker   <-chan time.Time
	position time.Duration
	duration time.Duration
}

// New returns a new mock TV.
func NewMock(root string) TV {
	m := &mockTV{
		ticker: time.Tick(time.Second),
	}
	go func() {
		for _ = range m.ticker {
			if m.playing != "" && !m.paused {
				m.position += time.Second
				if m.position >= m.duration {
					m.position = m.duration
				}
			}
		}
	}()
	return m
}

func (tv *mockTV) Playing() string {
	return tv.playing
}

func (tv *mockTV) Paused() bool {
	return tv.paused
}

func (tv *mockTV) CECErr() error {
	return nil
}

func (tv *mockTV) Play(filename string) error {
	tv.playing = filepath.Join(tv.root, filename)
	tv.paused = false
	tv.duration = (time.Duration(rand.Intn(1))*time.Hour +
		time.Duration(rand.Intn(59))*time.Minute +
		time.Duration(rand.Intn(59))*time.Second +
		time.Duration(rand.Intn(999))*time.Millisecond)
	return nil
}

func (tv *mockTV) Pause() error {
	tv.paused = !tv.paused
	return nil
}

func (tv *mockTV) Stop() error {
	tv.playing = ""
	tv.paused = false
	tv.position = 0
	tv.duration = 0
	return nil
}

func (tv *mockTV) Seek(d time.Duration) error {
	if tv.playing == "" {
		return nil
	}
	tv.position += d
	if tv.position < 0 {
		tv.position = 0
	}
	if tv.position >= tv.duration {
		tv.position = tv.duration
	}
	return nil
}

func (tv *mockTV) Position() time.Duration {
	return tv.position
}

func (tv *mockTV) Duration() time.Duration {
	return tv.duration
}
