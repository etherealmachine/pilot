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

func hasCookieOrPassword(r *http.Request) bool {
	if *password == "" {
		return true
	}
	cookie, err := r.Cookie("login")
	if err != nil {
		if *password == r.FormValue("password") {
			return true
	  }
		log.Printf("no login cookie or password: %v", err)
		return false
	}
	loginCookie := new(LoginCookie)
	if err = bakery.Decode("login", cookie.Value, loginCookie); err != nil {
		log.Printf("error decoding login cookie: %v", err)
		return false
	}
	return true
}
