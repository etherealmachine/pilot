package main

import (
	"fmt"
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

func loginRedirect(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, loginPage)
	return
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
	http.Redirect(w, r, r.FormValue("redirect_to"), http.StatusTemporaryRedirect)
}
