package main

import (
	"net/http"

	"github.com/etherealmachine/pilot/tv"
)

type Service struct {
	Files []string
	TV    *tv.TV
}

type ListFilesRequest struct {
}

type ListFilesResponse struct {
	Files []string
}

func (s *Service) ListFiles(r *http.Request, req *ListFilesRequest, resp *ListFilesResponse) error {
	resp.Files = s.Files
	return nil
}

type TVStatusRequest struct {
}

type TVStatusResponse struct {
	TV *tv.TV
}

func (s *Service) TVStatus(r *http.Request, req *TVStatusRequest, resp *TVStatusResponse) error {
	resp.TV = s.TV
	return nil
}

type TurnOnRequest struct {
}

type TurnOnResponse struct {
	TV *tv.TV
}

func (s *Service) TurnOn(r *http.Request, req *PlayRequest, resp *PlayResponse) error {
	err := s.TV.TurnOn()
	resp.TV = s.TV
	return err
}

type PlayRequest struct {
	Filename string
}

type PlayResponse struct {
	TV *tv.TV
}

func (s *Service) Play(r *http.Request, req *PlayRequest, resp *PlayResponse) error {
	err := s.TV.Play(req.Filename)
	resp.TV = s.TV
	return err
}

type PauseRequest struct {
}

type PauseResponse struct {
	TV *tv.TV
}

func (s *Service) Pause(r *http.Request, req *PauseRequest, resp *PauseResponse) error {
	err := s.TV.Pause()
	resp.TV = s.TV
	return err
}

type StopRequest struct {
}

type StopResponse struct {
	TV *tv.TV
}

func (s *Service) Stop(r *http.Request, req *StopRequest, resp *StopResponse) error {
	err := s.TV.Stop()
	resp.TV = s.TV
	return err
}
