package tv

import (
	"fmt"
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
	if rand.Float64() < 0.1 {
		return fmt.Errorf("random cec error")
	}
	return nil
}

func (tv *mockTV) Play(filename string) error {
	if rand.Float64() < 0.5 {
		return fmt.Errorf("random play error")
	}
	tv.playing = filepath.Join(tv.root, filename)
	tv.paused = false
	tv.duration = (time.Duration(rand.Intn(1))*time.Hour +
		time.Duration(rand.Intn(59))*time.Minute +
		time.Duration(rand.Intn(59))*time.Second +
		time.Duration(rand.Intn(999))*time.Millisecond)
	time.Sleep(2 * time.Second)
	return nil
}

func (tv *mockTV) Pause() error {
	if rand.Float64() < 0.5 {
		return fmt.Errorf("random pause error")
	}
	tv.paused = !tv.paused
	time.Sleep(2 * time.Second)
	return nil
}

func (tv *mockTV) Stop() error {
	tv.playing = ""
	tv.paused = false
	tv.position = 0
	tv.duration = 0
	if rand.Float64() < 0.5 {
		return fmt.Errorf("random stop error")
	}
	time.Sleep(2 * time.Second)
	return nil
}

func (tv *mockTV) Seek(d time.Duration) error {
	if tv.playing == "" {
		return nil
	}
	if rand.Float64() < 0.5 {
		return fmt.Errorf("random seek error")
	}
	tv.position += d
	if tv.position < 0 {
		tv.position = 0
	}
	if tv.position >= tv.duration {
		tv.position = tv.duration
	}
	time.Sleep(2 * time.Second)
	return nil
}

func (tv *mockTV) Position() time.Duration {
	return tv.position
}

func (tv *mockTV) Duration() time.Duration {
	return tv.duration
}
