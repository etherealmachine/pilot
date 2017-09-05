package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/etherealmachine/pilot/tv"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	root         = flag.String("root", ".", "Root folder to serve media from.")
	folders      = flag.String("folders", "TV,Movies", "Comma-separated list of folders to serve.")
	addr         = flag.String("addr", ":80", "Address to serve from.")
	password     = flag.String("password", "", "Login password.")
	logdir       = flag.String("logdir", "", "Location to save logs to. If empty, logs to stdout.")
	mocktv       = flag.Bool("mocktv", false, "Use mock TV for testing.")
	unvulcanized = flag.Bool("unvulcanized", false, "Serve unvulcanized app for testing.")
	filesjson    = flag.String("filesjson", "", "Serve given file as files.json for testing.")

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
	sync.RWMutex
	Files     []string
	filesHash string
	indexHash string
	TV        tv.TV
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
		if *password == "" {
			handler.ServeHTTP(w, r)
			return
		}
		if hasLoginCookie(r) || passwordProtectedDownload(r) {
			handler.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/" {
			s.LoginHandler(w, r)
			return
		} else {
			http.NotFound(w, r)
			return
		}
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

func (s *server) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	file, err := url.QueryUnescape(r.FormValue("file"))
	if err != nil {
		fmt.Fprintf(w, "error decoding query: %v", err)
		return
	}
	f, err := os.Open(filepath.Join(*root, file))
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
		"Content-Disposition", "attachment;filename="+filepath.Base(file))
	http.ServeContent(w, r, file, fi.ModTime(), f)
}

func (s *server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	pass := r.FormValue("password")
	if pass == *password {
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
		http.Redirect(w, r, "/", http.StatusFound)
	}
	w.WriteHeader(http.StatusFound)
	if err := loginTemplate.Execute(w, &struct {
		Error bool
	}{
		Error: pass != "",
	}); err != nil {
		log.Println(err)
	}
}

func (s *server) FilesHandler(w http.ResponseWriter, r *http.Request) {
	s.RLock()
	defer s.RUnlock()
	if *filesjson != "" {
		f, err := os.Open(*filesjson)
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
		http.ServeContent(w, r, *filesjson, fi.ModTime(), f)
		return
	}
	if h := r.Header.Get("If-None-Match"); h != "" && h == s.filesHash {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", s.filesHash)
	bs, err := json.Marshal(s.Files)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(bs)
}

func (s *server) FaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "favicon.ico")
}

func (s *server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	if h := r.Header.Get("If-None-Match"); h != "" && h == s.indexHash {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	bs, err := ioutil.ReadFile("index.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(bs)
		return
	}
	h := sha256.New()
	h.Write(bs)
	encodedBytes := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, encodedBytes)
	encoder.Write(h.Sum(nil))
	encoder.Close()
	s.indexHash = encodedBytes.String()
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "max-age=31536000")
	w.Header().Set("ETag", s.indexHash)
	w.Write(bs)
}

func (s *server) UnvulcanizedHandler(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/src") || strings.HasPrefix(r.URL.Path, "/bower_components") {
		http.ServeFile(w, r, filepath.Join("app", r.URL.Path))
		return
	}
	http.ServeFile(w, r, "app/index.html")
}

func main() {
	flag.Parse()

	logrus.SetLevel(logrus.DebugLevel)
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

	log.Println("pilot is up, looking for files to serve...")
	s.Lock()
	filepath.Walk(*root, walker(&s.Files))
	s.filesHash = calculateHash(s.Files)
	s.Unlock()
	log.Printf("found %d files", len(s.Files))

	http.Handle("/controls", rpcServer(s))
	http.HandleFunc("/download", s.DownloadHandler)
	http.HandleFunc("/files.json", s.FilesHandler)
	http.HandleFunc("/favicon.ico", s.FaviconHandler)
	if *unvulcanized {
		http.HandleFunc("/", s.UnvulcanizedHandler)
	} else {
		http.HandleFunc("/", s.IndexHandler)
	}

	log.Printf("Server listening at %s", *addr)
	log.Fatal(http.ListenAndServe(
		*addr,
		s.authenticate(logRequests(http.DefaultServeMux))))
}
