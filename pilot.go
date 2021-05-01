package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/etherealmachine/pilot/vlcctrl"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	root     = flag.String("root", ".", "Root folder to serve media from.")
	folders  = flag.String("folders", "TV,Movies", "Comma-separated list of folders to serve.")
	port     = flag.Int("port", 8080, "Port to serve from.")
	password = flag.String("password", "", "Login password.")
	logdir   = flag.String("logdir", "", "Location to save logs to. If empty, logs to stdout.")

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
	Player    vlcctrl.VLC
	Templates map[string]*template.Template
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

func (s *server) CurrentlyPlaying() string {
	playlist, err := s.Player.Playlist()
	if err != nil {
		log.Println(err)
		return ""
	}
	if len(playlist.Children) < 1 {
		return ""
	}
	if len(playlist.Children[0].Children) < 1 {
		return ""
	}
	return playlist.Children[0].Children[0].Name
}

func (s *server) PlayOnTV(filename string) error {
	fullpath := filepath.Join(*root, filename)
	log.Println("playing", fullpath)
	if err := s.Player.Stop(); err != nil {
		return err
	}
	if err := s.Player.EmptyPlaylist(); err != nil {
		return err
	}
	return s.Player.AddStart(fmt.Sprintf("file://%s", url.PathEscape(fullpath)))
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
	if err := s.Templates["login.html"].Execute(w, &struct {
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
	Playing string
	Movies  []string
	Shows   map[string]map[string][]string
	Filter  string
}

var re = regexp.MustCompile("[^a-z0-9]+")

func slugify(s ...string) string {
	var slugs []string
	for _, si := range s {
		slugs = append(slugs, strings.Trim(re.ReplaceAllString(strings.ToLower(si), "-"), "-"))
	}
	return strings.Join(slugs, "-")
}

func titleize(s string) string {
	title := filepath.Base(s)
	ext := filepath.Ext(title)
	return strings.TrimSuffix(title, ext)
}

func (p *IndexTemplateParams) InsertShow(show string, season string, episode string) {
	if p.Shows[show] == nil {
		p.Shows[show] = make(map[string][]string)
	}
	p.Shows[show][season] = append(p.Shows[show][season], episode)
}

func (s *server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	reload := r.URL.Query()["reload"]
	if len(reload) > 0 {
		s.reload()
	}
	params := &IndexTemplateParams{
		Playing: s.CurrentlyPlaying(),
		Shows:   make(map[string]map[string][]string),
		Filter:  "Movies",
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
	if err := s.Templates["index.html"].Execute(w, params); err != nil {
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
	if err := s.Templates["play.html"].Execute(w, params); err != nil {
		log.Println(err)
	}
}

type CastTemplateParams struct {
	Playing string
	UISrc   string
}

func (s *server) CastHandler(w http.ResponseWriter, r *http.Request) {
	var file string
	files := r.URL.Query()["file"]
	if len(files) > 0 {
		file = files[0]
	}
	if file != "" && file != s.CurrentlyPlaying() {
		s.PlayOnTV(file)
	}
	publicAddr := GetOutboundIP()
	params := &CastTemplateParams{
		Playing: s.CurrentlyPlaying(),
		UISrc:   fmt.Sprintf("http://%s:%d", publicAddr.String(), *port+1),
	}
	if err := s.Templates["cast.html"].Execute(w, params); err != nil {
		log.Println(err)
	}
}

func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func (s *server) reload() {
	var files []string
	filepath.Walk(*root, walker(&files))
	s.Lock()
	s.Files = files
	s.Unlock()
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
			MaxSize:  50, // megabytes MaxAge:   30, //days
		})
	} else {
		httplog = log.New(os.Stderr, "", log.Flags())
	}

	player, err := vlcctrl.NewVLC("127.0.0.1", 8081, "raspberry")
	if err != nil {
		log.Fatal(err)
	}
	if _, err = player.GetStatus(); err != nil {
		log.Fatal(fmt.Errorf("error, expected VLC on port 8081, got: %s", err))
	}

	s := &server{
		Templates: make(map[string]*template.Template),
		Player:    player,
	}
	for _, t := range []string{"index.html", "play.html", "login.html", "cast.html"} {
		s.Templates[t] = template.Must(template.New(t).Funcs(template.FuncMap{
			"slugify":    slugify,
			"titleize":   titleize,
			"trimPrefix": strings.TrimPrefix,
		}).ParseFiles(t))
	}

	log.Println("pilot is up, looking for files to serve...")
	s.Lock()
	filepath.Walk(*root, walker(&s.Files))
	s.Unlock()
	log.Printf("found %d files", len(s.Files))

	http.HandleFunc("/download", s.DownloadHandler)
	http.HandleFunc("/favicon.ico", s.FaviconHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/play", s.PlayHandler)
	http.HandleFunc("/cast", s.CastHandler)
	http.HandleFunc("/", s.IndexHandler)

	log.Fatal(http.ListenAndServe(
		fmt.Sprintf(":%d", *port),
		s.authenticate(logRequests(http.DefaultServeMux))))
}
