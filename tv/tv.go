package tv

import (
	"bufio"
	"io"
	"log"
	"os/exec"
)

const playerCommand = "/Users/james/Desktop/MPlayerX.app/Contents/MacOS/MPlayerX"

type TV struct {
	On       bool
	Playing  string
	player   *exec.Cmd
	playerIn io.WriteCloser
	cmdout   []string
	cmderr   []string
}

func New() *TV {
	return &TV{}
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

func (tv *TV) sendCEC(command string) error {
	cec := exec.Command("cec-client", "-s")
	tv.logCmd(cec)
	if err := cec.Start(); err != nil {
		return err
	}
	w, err := cec.StdinPipe()
	if err != nil {
		return err
	}
	defer w.Close()
	w.Write([]byte(command))
	return cec.Wait()
}

func (tv *TV) TurnOn() {
	if tv.On {
		return
	}
	if err := tv.sendCEC("on 0"); err != nil {
		log.Println(err)
	} else {
		tv.On = true
	}
}

func (tv *TV) TurnOff() {
	if !tv.On {
		return
	}
	if err := tv.sendCEC("standby 0"); err != nil {
		log.Println(err)
	} else {
		tv.On = false
	}
}

func (tv *TV) Play(filename string) error {
	if tv.player == nil {
		tv.player = exec.Command(playerCommand, filename)
		tv.logCmd(tv.player)
		if in, err := tv.player.StdinPipe(); err != nil {
			return err
		} else {
			tv.playerIn = in
		}
		if err := tv.player.Start(); err != nil {
			return err
		}
	}
	tv.Playing = filename
	return nil
}

func (tv *TV) Pause() error {
	if tv.player == nil {
		return nil
	}
	return exec.Command("xdotool", "key", "KP_Space").Run()
}

func (tv *TV) Stop() error {
	if tv.player == nil {
		return nil
	}
	if err := tv.player.Process.Kill(); err != nil {
		return err
	}
	tv.player = nil
	tv.Playing = ""
	return nil
}
