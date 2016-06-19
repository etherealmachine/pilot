package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
)

var bakery *securecookie.SecureCookie

var loginTemplate = template.Must(template.New("login").Parse(`<!doctype html>
<html>
	<head>
		<style>
			form {
				margin-top: 30px;
			}
			form {
				display: flex;
				justify-content: center;
			}
		</style>
	</head>
	<body>
		<div>
			<form
					action="/login"
					method="post">
	      <input
	      		type="password"
	      		name="password"
	      		placeholder="Password">
	      <button
	      		type="submit">
	      	Login
	      </button>
	      <input
	      		type="hidden"
	      		name="redirect_to"
	      		value="{{ .RedirectTo }}">
	    </form>
	   </div>
	</body>
</html>`))

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
