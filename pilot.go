package main

import (
	"bytes"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/etherealmachine/pilot/tv"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	root        = flag.String("root", ".", "Root folder to serve media from.")
	folders     = flag.String("folders", "TV,Movies", "Comma-separated list of folders to serve.")
	addr        = flag.String("addr", ":80", "Address to serve from.")
	build       = flag.Bool("build", false, "Build html file.")
	fromFS      = flag.Bool("fromfs", false, "Serve static content from the filesystem.")
	password    = flag.String("password", "", "Login password.")
	logfile     = flag.String("logfile", "", "Location to write logs to. If empty, logs to stdout.")
	startupTime time.Time
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

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if _, ok := getLoginCookie(r); !ok {
		loginRedirect(w, r)
		return
	}
	var filename string
	if r.URL.Path == "/" {
		filename = filepath.Join("static", "index.html")
	} else {
		filename = r.URL.Path[1:]
	}
	if *fromFS && strings.HasPrefix(filename, "static") {
		http.ServeFile(w, r, filename)
		return
	}
	if contents := static[filename]; contents != "" {
		http.ServeContent(w, r, filename, startupTime, bytes.NewReader([]byte(contents)))
		return
	}
	f, err := os.Open(filepath.Join(*root, filename))
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
		"Content-Disposition", "attachment;filename="+filepath.Base(filename))
	http.ServeContent(w, r, filename, fi.ModTime(), f)
}

func emptyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
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

func (s *Service) walk(path string, info os.FileInfo, _ error) error {
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

func main() {
	flag.Parse()

	startupTime = time.Now()

	if *logfile != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename: *logfile,
			MaxSize:  50, // megabytes
			MaxAge:   30, //days
		})
	}

	if *build {
		if err := buildStatic(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	svc := &Service{
		TV: tv.New(*root),
	}
	filepath.Walk(*root, svc.walk)
	log.Printf("found %d files", len(svc.Files))

	rpcserver := rpc.NewServer()
	rpcserver.RegisterCodec(json.NewCodec(), "application/json")
	rpcserver.RegisterService(svc, "Pilot")

	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/rpc", authWrap(rpcserver.ServeHTTP))
	http.HandleFunc("/null", emptyHandler)
	http.HandleFunc("/undefined", emptyHandler)
	http.HandleFunc("/", handleRoot)
	log.Printf("Server listening at %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, logRequests(http.DefaultServeMux)))
}
