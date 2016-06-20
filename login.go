package main

import (
	"html/template"
	"log"
	"net/http"
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

func getLoginCookie(r *http.Request) (*LoginCookie, bool) {
	if *password == "" {
		return &LoginCookie{}, true
	}
	cookie, err := r.Cookie("login")
	if err != nil {
		log.Printf("no login cookie: %v", err)
		return nil, false
	}
	loginCookie := new(LoginCookie)
	if err = bakery.Decode("login", cookie.Value, loginCookie); err != nil {
		log.Printf("error decoding login cookie: %v", err)
		return nil, false
	}
	return loginCookie, true
}
