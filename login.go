package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
)

var bakery *securecookie.SecureCookie

var loginTemplate = template.Must(template.ParseFiles("login.html"))

func init() {
	bakery = securecookie.New(
		securecookie.GenerateRandomKey(64),
		securecookie.GenerateRandomKey(32))
}

type LoginCookie struct {
	LoginTime time.Time
}

func hasLoginCookie(r *http.Request) bool {
	cookie, err := r.Cookie("login")
	if err != nil {
		return false
	}
	loginCookie := new(LoginCookie)
	if err = bakery.Decode("login", cookie.Value, loginCookie); err == nil {
		return true
	} else {
		log.Printf("error decoding login cookie: %v", err)
	}
	return false
}

func passwordProtectedDownload(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/download") && r.FormValue("password") == *password
}