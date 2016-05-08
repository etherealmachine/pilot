package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
)

var bakery *securecookie.SecureCookie

func init() {
	bakery = securecookie.New(
		securecookie.GenerateRandomKey(64),
		securecookie.GenerateRandomKey(32))
}

const loginPage = `<!doctype html>
<html>
	<head>
		<title>pilot</title>
	</head>
	<body>
		<form action="/login" method="post">
      <input id="password" type="password" name="password" placeholder="password">
      <button type="submit">Login</button>
    </form>
	</body>
</html>`

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
