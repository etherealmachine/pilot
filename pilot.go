package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/etherealmachine/pilot/tv"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	root     = flag.String("root", ".", "Root folder to serve media from.")
	folders  = flag.String("folders", "TV,Movies", "Comma-separated list of folders to serve.")
	addr     = flag.String("addr", ":80", "Address to serve from.")
	password = flag.String("password", "", "Login password.")
	logdir   = flag.String("logdir", "", "Location to save logs to. If empty, logs to stdout.")
	mocktv   = flag.Bool("mocktv", false, "Use mock TV for testing.")

	indexTemplate = template.Must(template.New("index.html").Funcs(template.FuncMap{
		"slugify":    slugify,
		"trimPrefix": strings.TrimPrefix,
	}).ParseFiles("index.html"))
	playTemplate = template.Must(template.New("play.html").Funcs(template.FuncMap{
		"slugify":    slugify,
		"trimPrefix": strings.TrimPrefix,
	}).ParseFiles("play.html"))

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

func (s *server) FaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "favicon.ico")
}

func (s *server) StaticHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "pure-min.css")
}

type IndexTemplateParams struct {
	Movies []string
	Shows  map[string]map[string][]string
	Filter string
}

var re = regexp.MustCompile("[^a-z0-9]+")

func slugify(s ...string) string {
	var slugs []string
	for _, si := range s {
		slugs = append(slugs, strings.Trim(re.ReplaceAllString(strings.ToLower(si), "-"), "-"))
	}
	return strings.Join(slugs, "-")
}

func (p *IndexTemplateParams) InsertShow(show string, season string, episode string) {
	if p.Shows[show] == nil {
		p.Shows[show] = make(map[string][]string)
	}
	p.Shows[show][season] = append(p.Shows[show][season], episode)
}

func (s *server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	params := &IndexTemplateParams{
		Shows:  make(map[string]map[string][]string),
		Filter: "Movies",
	}
	filters, ok := r.URL.Query()["filter"]
	if ok && len(filters) > 0 {
		params.Filter = filters[0]
	}
	for _, f := range s.Files {
		if !strings.HasPrefix(f, params.Filter) {
			continue
		}
		if strings.HasPrefix(f, "Movies") {
			params.Movies = append(params.Movies, f)
		} else if strings.HasPrefix(f, "TV") {
			path := strings.Split(f, "/")
			if len(path) == 3 {
				_, show, episode := path[0], path[1], path[2]
				params.InsertShow(show, "", episode)
			} else if len(path) == 4 {
				_, show, season, episode := path[0], path[1], path[2], path[3]
				params.InsertShow(show, season, episode)
			} else {
				panic(fmt.Sprintf("Failed to get episode information for %s", f))
			}
		}
	}
	if err := indexTemplate.Execute(w, params); err != nil {
		log.Println(err)
	}
}

type PlayTemplateParams struct {
	File  string
	Title string
}

func (s *server) PlayHandler(w http.ResponseWriter, r *http.Request) {
	file, ok := r.URL.Query()["file"]
	if !ok || len(file) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	title := filepath.Base(file[0])
	ext := filepath.Ext(title)
	params := &PlayTemplateParams{
		File:  file[0],
		Title: strings.TrimSuffix(title, ext),
	}
	if err := playTemplate.Execute(w, params); err != nil {
		log.Println(err)
	}
}

func (s *server) CastHandler(w http.ResponseWriter, r *http.Request) {
	file, ok := r.URL.Query()["file"]
	if !ok || len(file) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println(file[0])
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
		var err error
		s.TV, err = tv.New(*root)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("pilot is up, looking for files to serve...")
	s.Lock()
	filepath.Walk(*root, walker(&s.Files))
	s.filesHash = calculateHash(s.Files)
	s.Unlock()
	log.Printf("found %d files", len(s.Files))

	http.Handle("/controls", rpcServer(s))
	http.HandleFunc("/download", s.DownloadHandler)
	http.HandleFunc("/favicon.ico", s.FaviconHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/play", s.PlayHandler)
	http.HandleFunc("/cast", s.CastHandler)
	http.HandleFunc("/", s.IndexHandler)

	log.Printf("Server listening at %s", *addr)
	log.Fatal(http.ListenAndServe(
		*addr,
		s.authenticate(logRequests(http.DefaultServeMux))))
}
