package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
	logdir   = flag.String("logdir", "", "Location to save logs to. If empty, logs to stdout.")
	mocktv   = flag.Bool("mocktv", false, "Use mock TV for testing.")

	httplog *log.Logger
)

func logRequests(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httplog.Println(
			r.Host,
			r.RemoteAddr,
			r.Method,
			r.URL,
			r.Proto,
			r.Header.Get("User-Agent"))
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
	Files     []string
	filesHash string
	TV        tv.TV
	T         *template.Template
}

func walker(files *[]string) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, _ error) error {
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
			*files = append(*files, relPath)
		}
		return nil
	}
}

func (s *server) authenticate(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			if _, ok := getLoginCookie(r); !ok {
				t := s.T.Lookup("login.html")
				if err := t.Execute(w, &struct {
					RedirectTo string
				}{
					RedirectTo: r.URL.Path,
				}); err != nil {
					log.Println(err)
				}
				return
			}
		}
		handler.ServeHTTP(w, r)
	})
}

func calculateHash(files []string) string {
	h := sha256.New()
	for _, f := range files {
		h.Write([]byte(f))
	}
	encodedBytes := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, encodedBytes)
	encoder.Write(h.Sum(nil))
	encoder.Close()
	return encodedBytes.String()
}

func (s *server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	t := s.T.Lookup("index.html")
	if err := t.Execute(w, s); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (s *server) PlayHandler(w http.ResponseWriter, r *http.Request) {
	t := s.T.Lookup("play.html")
	if err := t.Execute(w, struct {
		Src string
	}{
		Src: r.FormValue("video"),
	}); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (s *server) ControlsHandler(w http.ResponseWriter, r *http.Request) {
	action := r.FormValue("action")
	var err error
	switch {
	case action == "play" && s.TV.Playing() == "":
		video := r.FormValue("video")
		found := false
		for _, f := range s.Files {
			if f == video {
				found = true
				break
			}
		}
		if !found {
			err = fmt.Errorf("no video named %q found", video)
			break
		}
		err = s.TV.Play(video)
	case (action == "pause" || action == "resume") && s.TV.Playing() != "":
		err = s.TV.Pause()
	case action == "stop":
		err = s.TV.Stop()
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
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
	if _, ok := getLoginCookie(r); ok {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
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
	http.ServeFile(w, r, "static/favicon.ico")
}

func (s *server) ReloadHandler(w http.ResponseWriter, r *http.Request) {
	var files []string
	filepath.Walk(*root, walker(&files))
	s.filesHash = calculateHash(files)
	s.Files = files
	fmt.Fprintf(w, "found %d files", len(s.Files))
}

func (s *server) FilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("If-None-Match") == s.filesHash {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Add("Cache-Control", "max-age=31536000")
	w.Header().Add("ETag", s.filesHash)
	bs, err := json.Marshal(s.Files)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(bs)
}

func main() {
	flag.Parse()

	log.SetFlags(log.Flags() | log.Lshortfile)
	if *logdir != "" {
		httplog = log.New(&lumberjack.Logger{
			Filename: filepath.Join(*logdir, "httprequests.log"),
			MaxSize:  50, // megabytes
			MaxAge:   30, //days
		}, "", log.Flags())
		log.SetOutput(&lumberjack.Logger{
			Filename: filepath.Join(*logdir, "output.log"),
			MaxSize:  50, // megabytes
			MaxAge:   30, //days
		})
	} else {
		httplog = log.New(os.Stderr, "", log.Flags())
	}

	s := &server{}
	if *mocktv {
		s.TV = tv.NewMock(*root)
	} else {
		s.TV = tv.New(*root)
	}

	var err error
	s.T, err = template.New("template").Funcs(template.FuncMap{
		"urlencode": url.QueryEscape,
	}).ParseGlob("static/*.html")
	if err != nil {
		log.Fatalf("error parsing templates: %v", err)
	}

	log.Println("pilot is up, looking for files to serve...")
	filepath.Walk(*root, walker(&s.Files))
	s.filesHash = calculateHash(s.Files)
	log.Printf("found %d files", len(s.Files))

	http.HandleFunc("/login", s.LoginHandler)
	http.HandleFunc("/play", s.PlayHandler)
	http.HandleFunc("/controls", s.ControlsHandler)
	http.HandleFunc("/download", s.DownloadHandler)
	http.HandleFunc("/favicon.ico", s.FaviconHandler)
	http.HandleFunc("/reload", s.ReloadHandler)
	http.HandleFunc("/files.json", s.FilesHandler)
	http.Handle("/js/", http.StripPrefix("/js", http.FileServer(http.Dir("static/js"))))
	http.Handle("/css/", http.StripPrefix("/css", http.FileServer(http.Dir("static/css"))))
	http.Handle("/fonts/", http.StripPrefix("/fonts", http.FileServer(http.Dir("static/fonts"))))
	http.HandleFunc("/", s.IndexHandler)

	log.Printf("Server listening at %s", *addr)
	log.Fatal(http.ListenAndServe(
		*addr,
		s.authenticate(logRequests(http.DefaultServeMux))))
}
