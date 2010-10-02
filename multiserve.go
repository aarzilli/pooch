package main

import (
	"http"
	"fmt"
	"strings"
)

func AddCookies(c http.ResponseWriter, cookies map[string]string) {
	for k, v := range cookies {
		c.SetHeader("Set-Cookie", fmt.Sprintf("%s=%s; Max-Age=65535; path=/", k, v))
	}
}

func GetCookies(c *http.Request) map[string]string {
	cookies := c.Header["Cookie"]
	cookiev := strings.Split(cookies, "=", 2)
	
	r := make(map[string]string)
	if len(cookiev) > 1 {
		r[cookiev[0]] = cookiev[1]
	}
	
	return r
}

func AddIdCookie(c http.ResponseWriter) {
	cookies := make(map[string]string)
	cookies["id"] = MakeRandomString(20);
	AddCookies(c, cookies);
}

func GetIdCookie(c *http.Request) string {
	cookies := GetCookies(c)
	return cookies["id"]
}

func LoginServer(c http.ResponseWriter, req *http.Request) {
	if req.FormValue("user") == "" {
		LoginHTML(map[string]string{ "problem": "" }, c)
	} else {
		//TODO:
		// - verificare username / password
		// - creare un cookie e associarlo nella tavola dei login e mostrare la pagina di successo
		// - se l'autenticazione fallisce mostrare la pagina di login
	}
}


func RegisterServer(c http.ResponseWriter, req *http.Request) {
	if req.FormValue("user") == "" {
		RegisterHTML(map[string]string{ "problem": "" }, c)
	} else {
		//TODO:
		// - verificare username / password
		// - se username non esiste registrarlo e andare alla pagina di login
		// - altrimenti mostrare la pagina di registrazione
	}
}

func MultiServe(port string) {
	http.HandleFunc("/login", WrapperServer(LoginServer))
	http.HandleFunc("/register", WrapperServer(RegisterServer))


	if err := http.ListenAndServe(":" + port, nil); err != nil {
		Log(ERROR, "Couldn't serve: ", err)
		return
	}
}