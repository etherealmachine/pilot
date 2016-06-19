package tv

import (
	"path/filepath"
)

type mockTV struct {
	root    string
	playing string
	paused  bool
}

// New returns a new mock TV.
func NewMock(root string) TV {
	return &mockTV{}
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
	return nil
}

func (tv *mockTV) Pause() error {
	tv.paused = !tv.paused
	return nil
}

func (tv *mockTV) Stop() error {
	tv.playing = ""
	tv.paused = false
	return nil
}

func (tv *mockTV) Seek(seconds int) error {
	return nil
}

func (tv *mockTV) Position() int64 {
	if tv.playing != "" {
		return 100000
	}
	return 0
}

func (tv *mockTV) Duration() int64 {
	if tv.playing != "" {
		return 1000000
	}
	return 0
}
