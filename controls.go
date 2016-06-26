package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
)

func rpcServer(s *server) *rpc.Server {
	rpcServer := rpc.NewServer()
	rpcServer.RegisterCodec(json.NewCodec(), "application/json")
	rpcServer.RegisterService(s, "Controls")
	return rpcServer
}

type EmptyRequest struct {
}

type StatusResponse struct {
	Playing  string `json:"playing"`
	Paused   bool   `json:"paused"`
	CECErr   error  `json:"cecErr"`
	Position int64  `json:"position"` // in milliseconds
	Duration int64  `json:"duration"` // in milliseconds
}

func (s *server) fillStatus(r *StatusResponse) {
	r.Playing = s.TV.Playing()
	r.Paused = s.TV.Paused()
	r.CECErr = s.TV.CECErr()
	r.Position = int64(s.TV.Position() / time.Millisecond)
	r.Duration = int64(s.TV.Duration() / time.Millisecond)
}

func (s *server) Status(r *http.Request, req *EmptyRequest, resp *StatusResponse) error {
	s.fillStatus(resp)
	return nil
}

type PlayRequest struct {
	File string `json:"file"`
}

func (s *server) Play(r *http.Request, req *PlayRequest, resp *StatusResponse) error {
	found := false
	for _, f := range s.Files {
		if f == req.File {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("no file named %q found", video)
	}
	err := s.TV.Play(req.File)
	s.fillStatus(resp)
	return err
}

func (s *server) Pause(r *http.Request, req *EmptyRequest, resp *StatusResponse) error {
	err := s.TV.Pause()
	s.fillStatus(resp)
	return err
}

func (s *server) Stop(r *http.Request, req *EmptyRequest, resp *StatusResponse) error {
	err := s.TV.Stop()
	s.fillStatus(resp)
	return err
}

type SeekRequest struct {
	Milliseconds int `json:"milliseconds"`
}

func (s *server) Seek(r *http.Request, req *SeekRequest, resp *StatusResponse) error {
	err := s.TV.Seek(time.Duration(req.Milliseconds) * time.Millisecond)
	s.fillStatus(resp)
	return err
}

type ReloadResponse struct {
	StatusResponse
	NumFiles int `json:"num_files"`
}

func (s *server) Reload(r *http.Request, req *EmptyRequest, resp *ReloadResponse) error {
	var files []string
	filepath.Walk(*root, walker(&files))
	s.filesHash = calculateHash(files)
	s.Files = files
	resp.NumFiles = len(s.Files)
	s.fillStatus(&resp.StatusResponse)
	if *mocktv {
		time.Sleep(2 * time.Second)
	}
	return nil
}
