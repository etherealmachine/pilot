package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/etherealmachine/pilot/tv"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	root     = flag.String("root", ".", "Root folder to serve media from.")
	folders  = flag.String("folders", "TV,Movies", "Comma-separated list of folders to serve.")
	addr     = flag.String("addr", ":80", "Address to serve from.")
	password = flag.String("password", "", "Login password.")
	logfile  = flag.String("logfile", "", "Location to write logs to. If empty, logs to stdout.")
)

func logRequests(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(
			r.Host,
			r.RemoteAddr,
			r.Method,
			r.URL,
			r.Proto,
			r.Header.Get("User-Agent"))
		handler.ServeHTTP(w, r)
	})
}

func authRequests(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			if _, ok := getLoginCookie(r); !ok {
				fmt.Fprint(w, loginPage)
				return
			}
		}
		handler.ServeHTTP(w, r)
	})
}

var video = map[string]bool{
	".mp4":  true,
	".avi":  true,
	".mpg":  true,
	".mov":  true,
	".wmv":  true,
	".mkv":  true,
	".m4v":  true,
	".webm": true,
	".flv":  true,
	".3gp":  true,
}

type server struct {
	Files []string
	TV    *tv.TV
	T     *template.Template
}

func (s *server) walk(path string, info os.FileInfo, _ error) error {
	if info.IsDir() {
		inFolder := path == *root
		for _, folder := range strings.Split(*folders, ",") {
			if strings.HasPrefix(path, filepath.Join(*root, folder)) {
				inFolder = true
				break
			}
		}
		if !inFolder {
			return filepath.SkipDir
		}
	}
	if video[filepath.Ext(path)] {
		relPath, err := filepath.Rel(*root, path)
		if err != nil {
			log.Printf("error scanning files: %v", err)
			return err
		}
		s.Files = append(s.Files, relPath)
	}
	return nil
}

func (s *server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	t := s.T.Lookup("index.html")
	if err := t.Execute(w, s); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (s *server) PlayHandler(w http.ResponseWriter, r *http.Request) {
	video, err := url.QueryUnescape(r.FormValue("video"))
	if err != nil {
		fmt.Fprintf(w, "error decoding query: %v", err)
		return
	}
	found := false
	for _, f := range s.Files {
		if f == video {
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(w, "no video named %q found", video)
		return
	}
	if r.FormValue("tv") == "true" {
		s.TV.Play(video)
		http.Redirect(w, r, "/controls", http.StatusTemporaryRedirect)
		return
	}
	t := s.T.Lookup("play.html")
	if err := t.Execute(w, struct {
		Src string
	}{
		Src: video,
	}); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (s *server) ControlsHandler(w http.ResponseWriter, r *http.Request) {
	action := r.FormValue("action")
	switch {
	case action == "resume" && s.TV.Playing != "" && s.TV.Paused:
		s.TV.Play(s.TV.Playing)
	case action == "pause" && s.TV.Playing != "" && !s.TV.Paused:
		s.TV.Pause()
	case action == "stop":
		s.TV.Stop()
	}

	t := s.T.Lookup("controls.html")
	if err := t.Execute(w, s); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (s *server) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	video, err := url.QueryUnescape(r.FormValue("video"))
	if err != nil {
		fmt.Fprintf(w, "error decoding query: %v", err)
		return
	}
	f, err := os.Open(filepath.Join(*root, video))
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	fi, err := f.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Add(
		"Content-Disposition", "attachment;filename="+filepath.Base(video))
	http.ServeContent(w, r, video, fi.ModTime(), f)
}

func (s *server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	pass := r.FormValue("password")
	if pass != *password {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	encoded, err := bakery.Encode("login", &LoginCookie{LoginTime: time.Now()})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "login",
		Value: encoded,
		Path:  "/",
	})
	http.Redirect(w, r, r.FormValue("redirect_to"), http.StatusTemporaryRedirect)
}

func (s *server) FaviconHandler(w http.ResponseWriter, r *http.Request) {
}

func (s *server) ReloadHandler(w http.ResponseWriter, r *http.Request) {
	s.Files = nil
	filepath.Walk(*root, s.walk)
	fmt.Fprintf(w, "found %d files", len(s.Files))
}

func main() {
	flag.Parse()

	if *logfile != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename: *logfile,
			MaxSize:  50, // megabytes
			MaxAge:   30, //days
		})
	}

	s := &server{
		TV: tv.New(*root),
	}

	var err error
	s.T, err = template.New("template").Funcs(template.FuncMap{
		"urlencode": url.QueryEscape,
	}).ParseGlob("templates/*.html")
	if err != nil {
		log.Fatalf("error parsing templates: %v", err)
	}

	filepath.Walk(*root, s.walk)
	log.Printf("found %d files", len(s.Files))

	http.HandleFunc("/login", s.LoginHandler)
	http.HandleFunc("/play", s.PlayHandler)
	http.HandleFunc("/controls", s.ControlsHandler)
	http.HandleFunc("/download", s.DownloadHandler)
	http.HandleFunc("/favicon.ico", s.FaviconHandler)
	http.HandleFunc("/reload", s.ReloadHandler)
	http.HandleFunc("/", s.IndexHandler)

	log.Printf("Server listening at %s", *addr)
	log.Fatal(http.ListenAndServe(
		*addr,
		authRequests(logRequests(http.DefaultServeMux))))
}
