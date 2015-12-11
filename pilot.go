package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"github.com/gorilla/securecookie"
)

var (
	root     = flag.String("root", ".", "Root folder to serve media from.")
	addr     = flag.String("addr", ":80", "Address to serve from.")
	build    = flag.Bool("build", false, "Build html file.")
	fromFS   = flag.Bool("fromfs", false, "Serve static content from the filesystem.")
	password = flag.String("password", "", "Login password.")

	bakery *securecookie.SecureCookie
	video  = map[string]bool{
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
)

type Service struct {
	Files []string
}

func (s *Service) walk(path string, _ os.FileInfo, _ error) error {
	if video[filepath.Ext(path)] {
		s.Files = append(s.Files, filepath.Clean(strings.TrimPrefix(path, *root)))
	}
	return nil
}

type LoginCookie struct {
	LoginTime time.Time
}

func getLoginCookie(r *http.Request) (*LoginCookie, bool) {
	if *password == "" {
		return &LoginCookie{}, true
	}
	cookie, err := r.Cookie("login")
	if err != nil {
		return nil, false
	}
	loginCookie := new(LoginCookie)
	if err = bakery.Decode("login", cookie.Value, loginCookie); err != nil {
		return nil, false
	}
	return loginCookie, true
}

func authWrap(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getLoginCookie(r); !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		f(w, r)
	}
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

func handleLogin(w http.ResponseWriter, r *http.Request) {
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
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	var filename string
	if r.URL.Path == "/" {
		if _, ok := getLoginCookie(r); !ok {
			filename = filepath.Join("static", "login.html")
		} else {
			filename = filepath.Join("static", "index.html")
		}
	} else {
		filename = r.URL.Path[1:]
	}
	if *fromFS {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			filename = filepath.Join(*root, r.URL.Path)
		}
		http.ServeFile(w, r, filename)
		return
	} else if contents := static[filename]; contents != "" {
		w.Write([]byte(contents))
		return
	}
	filename = filepath.Join(*root, filename)
	http.ServeFile(w, r, filename)
}

func emptyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	flag.Parse()

	if *build {
		if err := buildStatic(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	bakery = securecookie.New(
		securecookie.GenerateRandomKey(64),
		securecookie.GenerateRandomKey(32))

	svc := new(Service)
	go filepath.Walk(*root, svc.walk)
	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")
	server.RegisterService(svc, "Pilot")

	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/rpc", authWrap(server.ServeHTTP))
	http.HandleFunc("/null", emptyHandler)
	http.HandleFunc("/undefined", emptyHandler)
	http.HandleFunc("/", handleRoot)
	log.Printf("Server listening at %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
