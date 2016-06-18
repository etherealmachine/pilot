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
	tv.paused = true
	return nil
}

func (tv *mockTV) Unpause() error {
	tv.paused = false
	return nil
}

func (tv *mockTV) Stop() error {
	tv.playing = ""
	tv.paused = false
	return nil
}
